package cirrusenv

import (
	"bufio"
	"fmt"
	"github.com/google/uuid"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type CirrusEnv struct {
	file *os.File
}

func New() (*CirrusEnv, error) {
	cirrusEnvFile, err := os.Create(filepath.Join(os.TempDir(), "cirrus-env-"+uuid.New().String()))
	if err != nil {
		return nil, err
	}

	return &CirrusEnv{
		file: cirrusEnvFile,
	}, nil
}

func (ce *CirrusEnv) Path() string {
	return ce.file.Name()
}

func (ce *CirrusEnv) Consume() (map[string]string, error) {
	result := map[string]string{}

	if _, err := ce.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(ce.file)

	for scanner.Scan() {
		splits := strings.SplitN(scanner.Text(), "=", 2)
		if len(splits) != 2 {
			return nil, fmt.Errorf("CIRRUS_ENV file should contain lines in KEY=VALUE format")
		}

		result[splits[0]] = splits[1]
	}

	return result, nil
}

func (ce *CirrusEnv) Close() error {
	if err := ce.file.Close(); err != nil {
		return err
	}

	return os.Remove(ce.Path())
}

func Merge(sources ...map[string]string) map[string]string {
	destination := make(map[string]string)

	for _, source := range sources {
		for key, value := range source {
			destination[key] = value
		}
	}

	return destination
}
