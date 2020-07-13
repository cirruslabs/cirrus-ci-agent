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
	"io"
	"os"
	"path/filepath"
)

func UploadArtifacts(executor *Executor, name string, artifactsInstruction *api.CommandsResponse_ArtifactsInstruction, customEnv map[string]string) bool {
	logUploader, err := NewLogUploader(executor, name)
	if err != nil {
		request := api.ReportAgentProblemRequest{
			TaskIdentification: &executor.taskIdentification,
			Message:            fmt.Sprintf("Failed to initialize command clone log upload: %v", err),
		}
		_, _ = client.CirrusClient.ReportAgentWarning(context.Background(), &request)
		return false
	}
	defer logUploader.Finilize()

	uploadArtifactsClient, err := client.CirrusClient.UploadArtifacts(context.Background())
	if err != nil {
		_, _ = logUploader.Write([]byte(fmt.Sprintf("\nFailed to initialize artifacts upload client: %s", err)))
		return false
	}

	defer uploadArtifactsClient.CloseAndRecv()

	allAnnotations := make([]model.Annotation, 0)

	workingDir := customEnv["CIRRUS_WORKING_DIR"]

	for index, path := range artifactsInstruction.Paths {
		artifactsPattern := ExpandText(path, customEnv)
		artifactsPattern = filepath.Join(workingDir, artifactsPattern)
		artifactPaths, err := doublestar.Glob(artifactsPattern)

		if err != nil {
			_, _ = logUploader.Write([]byte(fmt.Sprintf("\nFailed to list artifacts: %s", err)))
			return false
		}

		if index > 0 {
			_, _ = logUploader.Write([]byte("\n"))
		}
		_, _ = logUploader.Write([]byte(fmt.Sprintf("Uploading %d artifacts for %s", len(artifactPaths), artifactsPattern)))

		chunkMsg := api.ArtifactEntry_ArtifactsUpload_{
			ArtifactsUpload: &api.ArtifactEntry_ArtifactsUpload{
				TaskIdentification: &executor.taskIdentification,
				Name:               name,
				Type:               artifactsInstruction.Type,
				Format:             artifactsInstruction.Format,
			},
		}
		err = uploadArtifactsClient.Send(&api.ArtifactEntry{Value: &chunkMsg})
		if err != nil {
			_, _ = logUploader.Write([]byte(fmt.Sprintf("\nFailed to initialize artifacts upload: %s", err)))
			return false
		}

		readBufferSize := int(1024 * 1024)
		readBuffer := make([]byte, readBufferSize)

		for _, artifactPath := range artifactPaths {
			if info, err := os.Stat(artifactPath); err == nil && info.IsDir() {
				continue
			}

			artifactFile, err := os.Open(artifactPath)
			if err != nil {
				_, _ = logUploader.Write([]byte(fmt.Sprintf("\nFailed to read artifact file %s: %s", artifactPath, err)))
				return false
			}
			//noinspection GoDeferInLoop
			defer artifactFile.Close()

			relativeArtifactPath, err := filepath.Rel(workingDir, artifactPath)
			if err != nil {
				_, _ = logUploader.Write([]byte(fmt.Sprintf("\nFailed to get artifact relative path for %s: %s", artifactPath, err)))
				return false
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
						_, _ = logUploader.Write([]byte(fmt.Sprintf("\nFailed to upload artifact file %s: %s", artifactPath, err)))
						return false
					}
					bytesUploaded += n
				}

				if err == io.EOF || n == 0 {
					break
				}
				if err != nil {
					_, _ = logUploader.Write([]byte(fmt.Sprintf("\nFailed to read artifact file %s: %s", artifactPath, err)))
					return false
				}
			}
			_, _ = logUploader.Write([]byte(fmt.Sprintf("\nUploaded %s", artifactPath)))

			if artifactsInstruction.Format != "" {
				_, _ = logUploader.Write([]byte(fmt.Sprintf("\nTrying to parse annotations for %s format", artifactsInstruction.Format)))
			}
			err, artifactAnnotations := annotations.ParseAnnotations(artifactsInstruction.Format, artifactPath)
			if err != nil {
				_, _ = logUploader.Write([]byte(fmt.Sprintf("\nFailed to create annotations: %s", err)))
				return false
			}
			allAnnotations = append(allAnnotations, artifactAnnotations...)
		}
	}

	if len(allAnnotations) > 0 {
		err := annotations.ValidateAnnotations(workingDir, allAnnotations)
		if err != nil {
			_, _ = logUploader.Write([]byte(fmt.Sprintf("\nFailed to validate annotations: %s", err)))
		}
		protoAnnotations := ConvertAnnotations(allAnnotations)
		reportAnnotationsCommandRequest := api.ReportAnnotationsCommandRequest{
			TaskIdentification: &executor.taskIdentification,
			Annotations:        protoAnnotations,
		}

		_, err = client.CirrusClient.ReportAnnotations(context.Background(), &reportAnnotationsCommandRequest)
		if err != nil {
			_, _ = logUploader.Write([]byte(fmt.Sprintf("\nFailed to report %d annotations: %s", len(allAnnotations), err)))
			_, _ = logUploader.Write([]byte(fmt.Sprintf("\nRetrying...")))
			_, err = client.CirrusClient.ReportAnnotations(context.Background(), &reportAnnotationsCommandRequest)
		}
		if err != nil {
			return false
		}
		_, _ = logUploader.Write([]byte(fmt.Sprintf("\nReported %d annotations!", len(allAnnotations))))
	}

	return true
}
