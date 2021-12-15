package executor

import (
	"bufio"
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/bmatcuk/doublestar"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"github.com/cirruslabs/cirrus-ci-annotations"
	"github.com/cirruslabs/cirrus-ci-annotations/model"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
)

type ProcessedArtifactPath struct {
	EnvExpandedPath string
	FinalPaths      []string
}

func (executor *Executor) UploadArtifacts(
	ctx context.Context,
	logUploader *LogUploader,
	name string,
	artifactsInstruction *api.ArtifactsInstruction,
	customEnv map[string]string,
) bool {
	var err error
	var allAnnotations []model.Annotation
	var scopedToWorkingDir bool

	if len(artifactsInstruction.Paths) == 0 {
		logUploader.Write([]byte("\nSkipping artifacts upload because there are no path specified..."))
		return true
	}

	err = retry.Do(
		func() error {
			allAnnotations, scopedToWorkingDir, err = executor.uploadArtifactsAndParseAnnotations(ctx, name,
				artifactsInstruction, customEnv, logUploader)
			return err
		}, retry.OnRetry(func(n uint, err error) {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to upload artifacts: %s", err)))
			logUploader.Write([]byte("\nRe-trying to upload artifacts..."))
		}),
		retry.Attempts(2),
		retry.Context(ctx),
	)
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to upload artifacts after multiple tries: %s", err)))
		return false
	}

	if len(allAnnotations) > 0 && scopedToWorkingDir {
		allAnnotations, err = annotations.NormalizeAnnotations(customEnv["CIRRUS_WORKING_DIR"], allAnnotations)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to validate annotations: %s", err)))
		}
		protoAnnotations := ConvertAnnotations(allAnnotations)
		reportAnnotationsCommandRequest := api.ReportAnnotationsCommandRequest{
			TaskIdentification: executor.taskIdentification,
			Annotations:        protoAnnotations,
		}

		err = retry.Do(
			func() error {
				_, err = client.CirrusClient.ReportAnnotations(ctx, &reportAnnotationsCommandRequest)
				return err
			}, retry.OnRetry(func(n uint, err error) {
				logUploader.Write([]byte(fmt.Sprintf("\nFailed to report %d annotations: %s", len(allAnnotations), err)))
				logUploader.Write([]byte("\nRetrying..."))
			}),
			retry.Attempts(2),
			retry.Context(ctx),
		)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nStill failed to report %d annotations: %s. Ignoring...", len(allAnnotations), err)))
			return true
		}
		logUploader.Write([]byte(fmt.Sprintf("\nReported %d annotations!", len(allAnnotations))))
	}

	return true
}

func (executor *Executor) uploadArtifactsAndParseAnnotations(
	ctx context.Context,
	name string,
	artifactsInstruction *api.ArtifactsInstruction,
	customEnv map[string]string,
	logUploader *LogUploader,
) ([]model.Annotation, bool, error) {
	allAnnotations := make([]model.Annotation, 0)

	uploadArtifactsClient, err := client.CirrusClient.UploadArtifacts(ctx)
	if err != nil {
		return allAnnotations, false, errors.Wrapf(err, "failed to initialize artifacts upload client")
	}

	defer func() {
		_, err := uploadArtifactsClient.CloseAndRecv()
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nError from upload stream: %s", err)))
		}
	}()

	workingDir := customEnv["CIRRUS_WORKING_DIR"]

	readBufferSize := int(1024 * 1024)
	readBuffer := make([]byte, readBufferSize)

	uploadSingleArtifactFile := func(scopedToWorkingDir bool, artifactPath string) error {
		artifactFile, err := os.Open(artifactPath)
		if err != nil {
			return errors.Wrapf(err, "failed to read artifact file %s", artifactPath)
		}
		defer artifactFile.Close()

		var relativeTo string
		if scopedToWorkingDir {
			relativeTo = workingDir
		} else {
			relativeTo = string(filepath.Separator)
		}

		relativeArtifactPath, err := filepath.Rel(relativeTo, artifactPath)
		if err != nil {
			return errors.Wrapf(err, "failed to get artifact relative path for %s", artifactPath)
		}

		bytesUploaded := 0
		bufferedFileReader := bufio.NewReaderSize(artifactFile, readBufferSize)

		for {
			n, err := bufferedFileReader.Read(readBuffer)

			if n > 0 {
				chunk := api.ArtifactEntry_ArtifactChunk{ArtifactPath: filepath.ToSlash(relativeArtifactPath), Data: readBuffer[:n]}
				chunkMsg := api.ArtifactEntry_Chunk{Chunk: &chunk}
				err := uploadArtifactsClient.Send(&api.ArtifactEntry{Value: &chunkMsg})
				if err != nil {
					return errors.Wrapf(err, "failed to upload artifact file %s", artifactPath)
				}
				bytesUploaded += n
			}

			if err == io.EOF || n == 0 {
				break
			}
			if err != nil {
				return errors.Wrapf(err, "failed to read artifact file %s", artifactPath)
			}
		}
		logUploader.Write([]byte(fmt.Sprintf("\nUploaded %s", artifactPath)))

		if artifactsInstruction.Format != "" {
			logUploader.Write([]byte(fmt.Sprintf("\nTrying to parse annotations for %s format", artifactsInstruction.Format)))
		}
		err, artifactAnnotations := annotations.ParseAnnotations(artifactsInstruction.Format, artifactPath)
		if err != nil {
			return errors.Wrapf(err, "failed to create annotations from %s", artifactPath)
		}
		allAnnotations = append(allAnnotations, artifactAnnotations...)
		return nil
	}

	// Process the paths specified by the user for this artifacts instruction
	var processedPaths []ProcessedArtifactPath

	for _, path := range artifactsInstruction.Paths {
		envExpandedPath := ExpandText(path, customEnv)

		var absoluteEnvExpandedPath string

		if filepath.IsAbs(envExpandedPath) {
			absoluteEnvExpandedPath = envExpandedPath
		} else {
			absoluteEnvExpandedPath = filepath.Join(workingDir, envExpandedPath)
		}

		finalPaths, err := doublestar.Glob(absoluteEnvExpandedPath)
		if err != nil {
			return allAnnotations, false, errors.Wrap(err, "Failed to list artifacts")
		}

		processedPaths = append(processedPaths, ProcessedArtifactPath{
			EnvExpandedPath: envExpandedPath,
			FinalPaths:      finalPaths,
		})
	}

	// Analyze processed paths for common denominator
	scopedToWorkingDir, err := isScopedToWorkingDir(workingDir, processedPaths)
	if err != nil {
		return allAnnotations, false, err
	}

	for index, processedPath := range processedPaths {
		if index > 0 {
			logUploader.Write([]byte("\n"))
		}
		logUploader.Write([]byte(fmt.Sprintf("Uploading %d artifacts for %s",
			len(processedPath.FinalPaths), processedPath.EnvExpandedPath)))

		chunkMsg := api.ArtifactEntry_ArtifactsUpload_{
			ArtifactsUpload: &api.ArtifactEntry_ArtifactsUpload{
				TaskIdentification: executor.taskIdentification,
				Name:               name,
				Type:               artifactsInstruction.Type,
				Format:             artifactsInstruction.Format,
			},
		}
		err = uploadArtifactsClient.Send(&api.ArtifactEntry{Value: &chunkMsg})
		if err != nil {
			return allAnnotations, scopedToWorkingDir, errors.Wrap(err, "failed to initialize artifacts upload")
		}

		for _, artifactPath := range processedPath.FinalPaths {
			info, err := os.Stat(artifactPath)

			if err == nil && info.IsDir() {
				logUploader.Write([]byte(fmt.Sprintf("\nSkipping uploading of '%s' because it's a folder", artifactPath)))
				continue
			}

			if err == nil && info.Size() > 100*humanize.MByte {
				humanFriendlySize := humanize.Bytes(uint64(info.Size()))
				logUploader.Write([]byte(fmt.Sprintf("\nUploading a quite hefty artifact '%s' of size %s",
					artifactPath, humanFriendlySize)))
			}

			err = uploadSingleArtifactFile(scopedToWorkingDir, artifactPath)

			if err != nil {
				return allAnnotations, scopedToWorkingDir, err
			}
		}
	}

	return allAnnotations, scopedToWorkingDir, nil
}

func isScopedToWorkingDir(workingDir string, processedPaths []ProcessedArtifactPath) (bool, error) {
	workingDirMatcher := filepath.Join(workingDir, "**")

	for _, processedPath := range processedPaths {
		for _, finalPath := range processedPath.FinalPaths {
			matched, err := doublestar.PathMatch(workingDirMatcher, finalPath)
			if err != nil {
				return false, errors.Wrapf(err, "Failed to match artifact paths against working directory matcher %v",
					workingDirMatcher)
			}
			if !matched {
				return false, nil
			}
		}
	}

	return true, nil
}
