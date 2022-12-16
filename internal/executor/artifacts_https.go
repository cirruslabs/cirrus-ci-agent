package executor

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"io"
	"net/http"
)

type HTTPSUploader struct {
	taskIdentification *api.TaskIdentification
	artifactName       string
	artifactType       string
	artifactFormat     string

	uploadedArtifactPaths []string
}

func NewHTTPSUploader(
	ctx context.Context,
	taskIdentification *api.TaskIdentification,
	artifactName string,
	artifactType string,
	artifactFormat string,
) (ArtifactUploader, error) {
	return &HTTPSUploader{
		taskIdentification: taskIdentification,
		artifactName:       artifactName,
		artifactType:       artifactType,
		artifactFormat:     artifactFormat,
	}, nil
}

func (uploader *HTTPSUploader) Upload(
	ctx context.Context,
	artifact io.Reader,
	relativeArtifactPath string,
) error {
	request := &api.GenerateArtifactUploadURLRequest{
		TaskIdentification: uploader.taskIdentification,
		Name:               uploader.artifactName,
		Path:               relativeArtifactPath,
	}

	response, err := client.CirrusClient.GenerateArtifactUploadURL(ctx, request)
	if err != nil {
		return err
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPut, response.Url, artifact)
	if err != nil {
		return err
	}

	for key, value := range response.Headers {
		httpRequest.Header.Set(key, value)
	}

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return err
	}

	if httpResponse.StatusCode != http.StatusOK && httpResponse.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to upload artifact file %s, HTTP status code: %d", relativeArtifactPath,
			httpResponse.StatusCode)
	}

	uploader.uploadedArtifactPaths = append(uploader.uploadedArtifactPaths, relativeArtifactPath)

	return nil
}

func (uploader *HTTPSUploader) Finish(ctx context.Context) error {
	if len(uploader.uploadedArtifactPaths) == 0 {
		return nil
	}

	_, err := client.CirrusClient.CommitUploadedArtifacts(ctx, &api.CommitUploadedArtifactsRequest{
		TaskIdentification: uploader.taskIdentification,
		Name:               uploader.artifactName,
		Type:               uploader.artifactType,
		Format:             uploader.artifactFormat,
		Paths:              uploader.uploadedArtifactPaths,
	})
	if err != nil {
		return err
	}

	return nil
}
