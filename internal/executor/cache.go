package executor

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/hasher"
	"github.com/cirruslabs/cirrus-ci-agent/internal/targz"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Cache struct {
	Name           string
	Key            string
	Folder         string
	FolderHash     string
	FilesHashes    map[string]string
	SkipUpload     bool
	CacheAvailable bool
}

var caches = make([]Cache, 0)

var httpClient = &http.Client{
	Timeout: time.Minute * 5,
}

func DownloadCache(executor *Executor, commandName string, cacheHost string, instruction *api.CacheInstruction, custom_env map[string]string) bool {
	logUploader, err := NewLogUploader(executor, commandName)
	if err != nil {
		return false
	}
	defer logUploader.Finalize()
	cacheKeyHash := sha256.New()

	if len(instruction.FingerprintScripts) > 0 {
		cmd, err := ShellCommandsAndWait(instruction.FingerprintScripts, &custom_env, func(bytes []byte) (int, error) {
			cacheKeyHash.Write(bytes)
			return logUploader.Write(bytes)
		}, &executor.timeout)
		if err != nil || !cmd.ProcessState.Success() {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to execute fingerprint script for %s cache!", commandName)))
			return false
		}
	} else {
		cacheKeyHash.Write([]byte(custom_env["CIRRUS_TASK_NAME"]))
		cacheKeyHash.Write([]byte(custom_env["CI_NODE_INDEX"]))
	}

	cacheKey := fmt.Sprintf("%s-%x", commandName, cacheKeyHash.Sum(nil))

	folderToCache := ExpandText(instruction.Folder, custom_env)

	if !filepath.IsAbs(folderToCache) {
		folderToCache = filepath.Join(custom_env["CIRRUS_WORKING_DIR"], folderToCache)
	}

	cachePopulated, cacheAvailable := tryToDownloadAndPopulateCache(logUploader, commandName, cacheHost, cacheKey, folderToCache)

	var folderToCacheHash = ""
	var folderToCacheFileHashes = make(map[string]string)
	if cachePopulated {
		folderToCacheHash, folderToCacheFileHashes, err = hasher.FolderHash(folderToCache)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to calculate hash of %s! %s", folderToCache, err)))
		}
	}

	if !cachePopulated && len(instruction.PopulateScripts) > 0 {
		logUploader.Write([]byte(fmt.Sprintf("\nCache miss for %s! Populating...\n", cacheKey)))
		cmd, err := ShellCommandsAndWait(instruction.PopulateScripts, &custom_env, func(bytes []byte) (int, error) {
			return logUploader.Write(bytes)
		}, &executor.timeout)
		if err != nil || cmd == nil || cmd.ProcessState == nil || !cmd.ProcessState.Success() {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to execute populate script for %s cache!", commandName)))
			return false
		}
	} else if !cachePopulated {
		logUploader.Write([]byte(fmt.Sprintf("\nCache miss for %s! No script to populate with.", cacheKey)))
	}

	caches = append(
		caches,
		Cache{
			Name:           commandName,
			Key:            cacheKey,
			Folder:         folderToCache,
			FolderHash:     folderToCacheHash,
			FilesHashes:    folderToCacheFileHashes,
			SkipUpload:     cacheAvailable && !instruction.ReuploadOnChanges,
			CacheAvailable: cacheAvailable,
		},
	)
	return true
}

func tryToDownloadAndPopulateCache(
	logUploader *LogUploader,
	commandName string,
	cacheHost string,
	cacheKey string,
	folderToCache string,
) (bool, bool) { // successfully populated, available remotely
	cacheFile, err := FetchCache(logUploader, commandName, cacheHost, cacheKey)
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to fetch archive for %s cache: %s!", commandName, err)))
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return false, true
		} else {
			return false, false
		}
	}
	if cacheFile == nil {
		return false, false
	}
	_, _ = logUploader.Write([]byte(fmt.Sprintf("\nCache hit for %s!", cacheKey)))
	unarchiveStartTime := time.Now()
	err = unarchiveCache(cacheFile, folderToCache)
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to unarchive %s cache because of %s! Retrying...\n", commandName, err)))
		os.RemoveAll(folderToCache)
		cacheFile, err := FetchCache(logUploader, commandName, cacheHost, cacheKey)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to fetch archive for %s cache: %s!", commandName, err)))
			if err, ok := err.(net.Error); ok && err.Timeout() {
				return false, true
			} else {
				return false, false
			}
		}
		if cacheFile == nil {
			return false, true
		}
		err = unarchiveCache(cacheFile, folderToCache)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed again to unarchive %s cache because of %s!\n", commandName, err)))
			logUploader.Write([]byte(fmt.Sprintf("\nTreating this failure as a cache miss but won't try to re-upload! Cleaning up %s...\n", folderToCache)))
			os.RemoveAll(folderToCache)
			EnsureFolderExists(folderToCache)
			return false, true
		}
	} else {
		unarchiveDuration := time.Since(unarchiveStartTime)
		if unarchiveDuration > 10*time.Second {
			logUploader.Write([]byte(fmt.Sprintf("\nUnarchived %s cache entry in %f seconds!\n", commandName, unarchiveDuration.Seconds())))
		}
	}
	return true, true
}

func unarchiveCache(
	cacheFile *os.File,
	folderToCache string,
) error {
	defer os.Remove(cacheFile.Name())
	EnsureFolderExists(folderToCache)
	return targz.Unarchive(cacheFile.Name(), folderToCache)
}

func FetchCache(logUploader *LogUploader, commandName string, cacheHost string, cacheKey string) (*os.File, error) {
	cacheFile, err := ioutil.TempFile(os.TempDir(), commandName)
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nCache miss for %s!", commandName)))
		return nil, err
	}
	defer cacheFile.Close()

	httpClient := http.Client{
		Timeout: 5 * time.Minute,
	}
	downloadStartTime := time.Now()
	resp, err := httpClient.Get(fmt.Sprintf("http://%s/%s", cacheHost, cacheKey))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	bufferedFileWriter := bufio.NewWriter(cacheFile)
	bytesDownloaded, err := bufferedFileWriter.ReadFrom(bufio.NewReader(resp.Body))
	if err != nil {
		return nil, err
	}
	err = bufferedFileWriter.Flush()
	if err != nil {
		return nil, err
	}
	downloadDuration := time.Since(downloadStartTime)
	if bytesDownloaded < 1024 {
		logUploader.Write([]byte(fmt.Sprintf("\nDownloaded %d bytes.", bytesDownloaded)))
	} else if bytesDownloaded < 1024*1024 {
		logUploader.Write([]byte(fmt.Sprintf("\nDownloaded %dKb.", bytesDownloaded/1024)))
	} else {
		logUploader.Write([]byte(fmt.Sprintf("\nDownloaded %dMb in %fs.", bytesDownloaded/1024/1024, downloadDuration.Seconds())))
	}
	return cacheFile, nil
}

func UploadCache(executor *Executor, commandName string, cacheHost string, instruction *api.UploadCacheInstruction) bool {
	logUploader, err := NewLogUploader(executor, commandName)
	if err != nil {
		return false
	}
	defer logUploader.Finalize()

	cache := FindCache(instruction.CacheName)

	if cache == nil {
		logUploader.Write([]byte(fmt.Sprintf("No cache found for %s!", instruction.CacheName)))
		return false // cache record should always exists
	}

	if cache.SkipUpload {
		logUploader.Write([]byte(fmt.Sprintf("Skipping change detection for %s cache!", instruction.CacheName)))
		return true
	}

	if isDirEmpty(cache.Folder) {
		logUploader.Write([]byte(fmt.Sprintf("Folder %s is empty! Skipping uploading ...", cache.Folder)))
		return true
	}

	folderToCacheHash, folderToCacheFileHashes, err := hasher.FolderHash(cache.Folder)
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("Failed to calculate hash of %s! %s", cache.Folder, err)))
		logUploader.Write([]byte(fmt.Sprintf("Skipping uploading of %s!", cache.Folder)))
		return true
	}

	logUploader.Write([]byte(fmt.Sprintf("SHA for %s is '%s'\n", cache.Folder, folderToCacheHash)))

	if folderToCacheHash == cache.FolderHash {
		logUploader.Write([]byte(fmt.Sprintf("Cache %s hasn't changed! Skipping uploading...", cache.Name)))
		return true
	}
	if cache.FolderHash != "" {
		logUploader.Write([]byte(fmt.Sprintf("Cache %s has changed!", cache.Name)))
		logUploader.Write([]byte(fmt.Sprintf("\nList of changes for %s:", cache.Folder)))
		for endFilePath, endFileHash := range folderToCacheFileHashes {
			startFileHash, ok := cache.FilesHashes[endFilePath]
			if !ok {
				logUploader.Write([]byte(fmt.Sprintf("\ncreated: %s", endFilePath)))
			} else if endFileHash != startFileHash {
				logUploader.Write([]byte(fmt.Sprintf("\nmodified: %s", endFilePath)))
			}
		}
		for startFilePath := range cache.FilesHashes {
			_, ok := folderToCacheFileHashes[startFilePath]
			if !ok {
				logUploader.Write([]byte(fmt.Sprintf("\ndeleted: %s", startFilePath)))
			}
		}
	}

	cacheFile, _ := ioutil.TempFile(os.TempDir(), cache.Key)
	defer os.Remove(cacheFile.Name())

	err = targz.Archive(cache.Folder, cacheFile.Name())
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to tar caches for %s with %s!", commandName, err)))
		return false
	}
	fi, err := cacheFile.Stat()
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to create caches archive for %s with %s!", commandName, err)))
		return false
	}

	bytesToUpload := fi.Size()
	if bytesToUpload >= 2*1000*1000*1000 {
		logUploader.Write([]byte(fmt.Sprintf("\nCache %s is too big! Skipping caching...", commandName)))
		return true
	}

	if bytesToUpload < 1024 {
		logUploader.Write([]byte(fmt.Sprintf("\n%s cache size is %d bytes.", instruction.CacheName, bytesToUpload)))
	} else if bytesToUpload < 1024*1024 {
		logUploader.Write([]byte(fmt.Sprintf("\n%s cache size is %dKb.", instruction.CacheName, bytesToUpload/1024)))
	} else {
		logUploader.Write([]byte(fmt.Sprintf("\n%s cache size is %dMb.", instruction.CacheName, bytesToUpload/1024/1024)))
	}

	if !cache.CacheAvailable {
		// check if some other task has uploaded the cache already
		response, _ := httpClient.Head(fmt.Sprintf("http://%s/%s", cacheHost, cache.Key))
		if response != nil && response.StatusCode == http.StatusOK {
			logUploader.Write([]byte(fmt.Sprintf("\nSome other task has already uploaded cache entry %s! Skipping upload...", cache.Key)))
			return true
		}
	}

	logUploader.Write([]byte(fmt.Sprintf("\nUploading cache %s...", instruction.CacheName)))
	err = UploadCacheFile(cacheHost, cache.Key, cacheFile)
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to upload cache '%s': %s!", commandName, err)))
		logUploader.Write([]byte("\nIgnoring the error..."))
		return true
	}

	return true
}

func UploadCacheFile(cacheHost string, cacheKey string, cacheFile *os.File) error {
	response, err := httpClient.Post(
		fmt.Sprintf("http://%s/%s", cacheHost, cacheKey),
		"application/octet-stream",
		cacheFile,
	)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response status from HTTP cache %d: %s", response.StatusCode, response.Status)
	}
	return nil
}

func FindCache(cacheName string) *Cache {
	for i := 0; i < len(caches); i++ {
		if caches[i].Name == cacheName {
			return &caches[i]
		}
	}
	return nil
}
