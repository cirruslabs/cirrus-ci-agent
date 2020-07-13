package hasher

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func FolderHash(folderPath string) (string, map[string]string, error) {
	folderHash := sha256.New()
	fileHashes := make(map[string]string)
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		return "", fileHashes, nil
	}
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		fileHash, err := FileHash(path)
		// symlink can still be a directory
		if err != nil && strings.Contains(err.Error(), "is a directory") {
			return nil
		}
		if err != nil && os.IsNotExist(err) && (info.Mode()&os.ModeSymlink != 0) {
			destination, linkErr := os.Readlink(path)
			if linkErr == nil {
				hasher := sha256.New()
				_, err = hasher.Write([]byte(destination))
				fileHash = hasher.Sum(nil)
			}
		}
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(folderPath, path)
		if err != nil {
			return err
		}
		fileHashes[relativePath] = fmt.Sprintf("%x", fileHash)
		_, err = folderHash.Write(fileHash)
		return err
	})
	if err != nil {
		return "", fileHashes, err
	}
	return fmt.Sprintf("%x", folderHash.Sum(nil)), fileHashes, nil
}

func FileHash(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	shaHash := sha256.New()
	_, err = io.Copy(shaHash, f)
	if err != nil {
		return nil, err
	}
	return shaHash.Sum(nil), nil
}
