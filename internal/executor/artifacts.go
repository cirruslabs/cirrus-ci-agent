package executor

import (
	"bufio"
	"context"
	"fmt"
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

func UploadArtifacts(executor *Executor, name string, artifactsInstruction *api.ArtifactsInstruction, customEnv map[string]string) bool {
	logUploader, err := NewLogUploader(executor, name, customEnv)
	if err != nil {
		request := api.ReportAgentProblemRequest{
			TaskIdentification: executor.taskIdentification,
			Message:            fmt.Sprintf("Failed to initialize command clone log upload: %v", err),
		}
		client.CirrusClient.ReportAgentWarning(context.Background(), &request)
		return false
	}
	defer logUploader.Finalize()

	allAnnotations, err := uploadArtifactsAndParseAnnotations(executor, name, artifactsInstruction, customEnv, logUploader)
	if err != nil {
		logUploader.Write([]byte(fmt.Sprintf("\nFailed to upload artifacts: %s", err)))
		logUploader.Write([]byte("\nRe-trying to upload artifacts..."))

		allAnnotations, err = uploadArtifactsAndParseAnnotations(executor, name, artifactsInstruction, customEnv, logUploader)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to upload artifacts again: %s", err)))
			return false
		}
	}

	workingDir := customEnv["CIRRUS_WORKING_DIR"]
	if len(allAnnotations) > 0 {
		err := annotations.ValidateAnnotations(workingDir, allAnnotations)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to validate annotations: %s", err)))
		}
		protoAnnotations := ConvertAnnotations(allAnnotations)
		reportAnnotationsCommandRequest := api.ReportAnnotationsCommandRequest{
			TaskIdentification: executor.taskIdentification,
			Annotations:        protoAnnotations,
		}

		_, err = client.CirrusClient.ReportAnnotations(context.Background(), &reportAnnotationsCommandRequest)
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nFailed to report %d annotations: %s", len(allAnnotations), err)))
			logUploader.Write([]byte("\nRetrying..."))
			_, err = client.CirrusClient.ReportAnnotations(context.Background(), &reportAnnotationsCommandRequest)
		}
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nStill failed to report %d annotations: %s. Ignoring...", len(allAnnotations), err)))
			return true
		}
		logUploader.Write([]byte(fmt.Sprintf("\nReported %d annotations!", len(allAnnotations))))
	}

	return true
}

func uploadArtifactsAndParseAnnotations(
	executor *Executor,
	name string,
	artifactsInstruction *api.ArtifactsInstruction,
	customEnv map[string]string,
	logUploader *LogUploader,
) ([]model.Annotation, error) {
	allAnnotations := make([]model.Annotation, 0)

	uploadArtifactsClient, err := client.CirrusClient.UploadArtifacts(context.Background())
	if err != nil {
		return allAnnotations, errors.Wrapf(err, "failed to initialize artifacts upload client")
	}

	defer func() {
		_, err := uploadArtifactsClient.CloseAndRecv()
		if err != nil {
			logUploader.Write([]byte(fmt.Sprintf("\nError from upload stream: %s", err)))
		}
	}()

	workingDir := customEnv["CIRRUS_WORKING_DIR"]

	for index, path := range artifactsInstruction.Paths {
		artifactsPattern := ExpandText(path, customEnv)
		artifactsPattern = filepath.Join(workingDir, artifactsPattern)
		artifactPaths, err := doublestar.Glob(artifactsPattern)

		if err != nil {
			return allAnnotations, errors.Wrap(err, "Failed to list artifacts")
		}

		if index > 0 {
			logUploader.Write([]byte("\n"))
		}
		logUploader.Write([]byte(fmt.Sprintf("Uploading %d artifacts for %s", len(artifactPaths), artifactsPattern)))

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
			return allAnnotations, errors.Wrap(err, "failed to initialize artifacts upload")
		}

		readBufferSize := int(1024 * 1024)
		readBuffer := make([]byte, readBufferSize)

		for _, artifactPath := range artifactPaths {
			info, err := os.Stat(artifactPath)

			if err == nil && info.IsDir() {
				continue
			}

			if err == nil && info.Size() > 100*humanize.MByte {
				humanFriendlySize := humanize.Bytes(uint64(info.Size()))
				logUploader.Write([]byte(fmt.Sprintf("Uploading a quite hefty artifact '%s' of size %s",
					artifactPath, humanFriendlySize)))
			}

			artifactFile, err := os.Open(artifactPath)
			if err != nil {
				return allAnnotations, errors.Wrapf(err, "failed to read artifact file %s", artifactPath)
			}
			//noinspection GoDeferInLoop
			defer artifactFile.Close()

			relativeArtifactPath, err := filepath.Rel(workingDir, artifactPath)
			if err != nil {
				return allAnnotations, errors.Wrapf(err, "failed to get artifact relative path for %s", artifactPath)
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
						return allAnnotations, errors.Wrapf(err, "failed to upload artifact file %s", artifactPath)
					}
					bytesUploaded += n
				}

				if err == io.EOF || n == 0 {
					break
				}
				if err != nil {
					return allAnnotations, errors.Wrapf(err, "failed to read artifact file %s", artifactPath)
				}
			}
			logUploader.Write([]byte(fmt.Sprintf("\nUploaded %s", artifactPath)))

			if artifactsInstruction.Format != "" {
				logUploader.Write([]byte(fmt.Sprintf("\nTrying to parse annotations for %s format", artifactsInstruction.Format)))
			}
			err, artifactAnnotations := annotations.ParseAnnotations(artifactsInstruction.Format, artifactPath)
			if err != nil {
				return allAnnotations, errors.Wrapf(err, "failed to create annotations from %s", artifactPath)
			}
			allAnnotations = append(allAnnotations, artifactAnnotations...)
		}
	}
	return allAnnotations, nil
}
