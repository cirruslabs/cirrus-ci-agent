package executor

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"github.com/bmatcuk/doublestar"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/hasher"
	"github.com/cirruslabs/cirrus-ci-agent/internal/targz"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Cache struct {
	Name           string
	Key            string
	BaseFolder     string
	FoldersToCache []string
	Glob           string
	FileHasher     *hasher.Hasher
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

	folderToCache, err = filepath.Abs(folderToCache)
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to compute absolute path for cache folder '%s': %s\n",
			folderToCache, err)))
		return false
	}

	var glob string
	baseFolder := folderToCache
	if pathLooksLikeGlob(folderToCache) {
		baseFolder = custom_env["CIRRUS_WORKING_DIR"]
		glob = folderToCache

		// Sanity check
		//
		// When glob is used, the semantics are clearly defined only when
		// it's matches are scoped to the current working directory,
		// because otherwise it's impossible to make paths inside of the
		// archive portable (i.e. independent on the location of the working
		// directory).
		//
		// Note: this is not a security stop-gap but merely a hint to the users
		// that they are doing something potentially wrong.
		terminatedWorkingDir := custom_env["CIRRUS_WORKING_DIR"]

		if !strings.HasSuffix(terminatedWorkingDir, string(os.PathSeparator)) {
			terminatedWorkingDir += string(os.PathSeparator)
		}

		if !strings.HasPrefix(glob, terminatedWorkingDir) {
			logUploader.Write([]byte(fmt.Sprintf("\nCannot expand cache folder glob '%s' that points above the current working directory %s\n",
				glob, terminatedWorkingDir)))
			return false
		}
	}

	cachePopulated, cacheAvailable := tryToDownloadAndPopulateCache(logUploader, commandName, cacheHost, cacheKey, baseFolder)

	foldersToCache := []string{folderToCache}
	if glob != "" {
		// Expand the glob so we can calculate the hashes for directories that already exist
		foldersToCache, err = doublestar.Glob(glob)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nCannot expand cache folder glob '%s': %s\n", glob, err)))
			return false
		}
	}

	fileHasher := hasher.New()
	if cachePopulated {
		for _, folderToCache := range foldersToCache {
			if err := fileHasher.AddFolder(baseFolder, folderToCache); err != nil {
				logUploader.Write([]byte(fmt.Sprintf("\nFailed to calculate hash of %s! %s", folderToCache, err)))
			}
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
			BaseFolder:     baseFolder,
			FoldersToCache: foldersToCache,
			Glob:           glob,
			FileHasher:     fileHasher,
			SkipUpload:     cacheAvailable && !instruction.ReuploadOnChanges,
			CacheAvailable: cacheAvailable,
		},
	)
	return true
}

func pathLooksLikeGlob(path string) bool {
	return strings.Contains(path, "*")
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

	if cache.Glob != "" {
		// Expand the glob again to capture the folders that were created after the first expansion in DownloadCache()
		cache.FoldersToCache, err = doublestar.Glob(cache.Glob)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nCannot expand cache folder glob '%s': %s\n", cache.Glob, err)))
			return false
		}
	}

	commaSeparatedFolders := strings.Join(cache.FoldersToCache, ", ")

	if allDirsEmpty(cache.FoldersToCache) {
		logUploader.Write([]byte(fmt.Sprintf("All cache folders (%s) are empty! Skipping uploading ...", commaSeparatedFolders)))
		return true
	}

	fileHasher := hasher.New()
	for _, folder := range cache.FoldersToCache {
		if err := fileHasher.AddFolder(cache.BaseFolder, folder); err != nil {
			logUploader.Write([]byte(fmt.Sprintf("Failed to calculate hash of %s! %s", folder, err)))
			logUploader.Write([]byte("Skipping uploading of cache!"))
			return true
		}
	}

	logUploader.Write([]byte(fmt.Sprintf("SHA for cache folders (%s) is '%s'\n", commaSeparatedFolders, fileHasher.SHA())))

	if fileHasher.SHA() == cache.FileHasher.SHA() {
		logUploader.Write([]byte(fmt.Sprintf("Cache %s hasn't changed! Skipping uploading...", cache.Name)))
		return true
	}
	if cache.FileHasher.Len() != 0 {
		logUploader.Write([]byte(fmt.Sprintf("Cache %s has changed!", cache.Name)))
		logUploader.Write([]byte(fmt.Sprintf("\nList of changes for cache folders (%s):", commaSeparatedFolders)))

		for _, diffEntry := range cache.FileHasher.DiffWithNewer(fileHasher) {
			logUploader.Write([]byte(fmt.Sprintf("\n%s: %s", diffEntry.Type.String(), diffEntry.Path)))
		}
	}

	cacheFile, _ := ioutil.TempFile(os.TempDir(), cache.Key)
	defer os.Remove(cacheFile.Name())

	err = targz.Archive(cache.BaseFolder, cache.FoldersToCache, cacheFile.Name())
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
