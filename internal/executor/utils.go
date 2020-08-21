package executor

import (
	"encoding/hex"
	"errors"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-annotations/model"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

func TempFileName(prefix, suffix string) (*os.File, error) {
	randBytes := make([]byte, 16)
	for i := 0; i < 10000; i++ {
		rand.Read(randBytes)
		path := filepath.Join(os.TempDir(), prefix+hex.EncodeToString(randBytes)+suffix)
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			continue
		}
		return f, err
	}
	return nil, errors.New("failed to create temp file")
}

func EnsureFolderExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			log.Printf("Failed to mkdir %s: %s", path, err)
		}
	}
}

func isDirEmpty(path string) bool {
	files, err := ioutil.ReadDir(path)
	if os.IsNotExist(err) {
		return true
	}
	if err != nil {
		return false
	}
	return len(files) == 0
}

func ConvertAnnotations(annotations []model.Annotation) []*api.Annotation {
	result := make([]*api.Annotation, 0)
	for _, annotation := range annotations {
		protoAnnotation := api.Annotation{
			Type:               api.Annotation_Type(annotation.Type),
			Level:              api.Annotation_Level(api.Annotation_Level_value[strings.ToUpper(annotation.Level)]),
			Message:            annotation.Message,
			RawDetails:         annotation.RawDetails,
			FullyQualifiedName: annotation.FullyQualifiedName,
		}
		if annotation.Location != nil {
			protoAnnotation.FileLocation = &api.Annotation_FileLocation{
				Path:        annotation.Location.Path,
				StartLine:   annotation.Location.StartLine,
				EndLine:     annotation.Location.EndLine,
				StartColumn: annotation.Location.StartColumn,
				EndColumn:   annotation.Location.EndColumn,
			}
		}
		result = append(result, &protoAnnotation)
	}
	return result
}
