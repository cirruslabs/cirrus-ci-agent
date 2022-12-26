package executor

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"io"
	"net/http"
)

type UploadDescriptor struct {
	url     string
	headers map[string]string
}

type HTTPSUploader struct {
	taskIdentification *api.TaskIdentification

	artifacts         *Artifacts
	uploadDescriptors map[string]*UploadDescriptor
}

func NewHTTPSUploader(
	ctx context.Context,
	taskIdentification *api.TaskIdentification,
	artifacts *Artifacts,
) (ArtifactUploader, error) {
	// Generate URLs to which we'll upload the artifacts
	request := &api.GenerateArtifactUploadURLsRequest{
		TaskIdentification: taskIdentification,
		Name:               artifacts.Name,
		Paths:              artifacts.UploadableRelativePaths(),
	}

	response, err := client.CirrusClient.GenerateArtifactUploadURLs(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(request.Paths) != len(response.Urls) {
		return nil, fmt.Errorf("GenerateArtifactUploadURLs() RPC call returned invalid data:"+
			" requested %d URLs, got %d", len(request.Paths), len(response.Urls))
	}

	// Create a mapping between relative artifact paths and upload URLs
	uploadDescriptors := map[string]*UploadDescriptor{}

	for idx, url := range response.Urls {
		uploadDescriptors[request.Paths[idx]] = &UploadDescriptor{
			url:     url.Url,
			headers: url.Headers,
		}
	}

	return &HTTPSUploader{
		taskIdentification: taskIdentification,
		artifacts:          artifacts,
		uploadDescriptors:  uploadDescriptors,
	}, nil
}

func (uploader *HTTPSUploader) Upload(
	ctx context.Context,
	artifact io.Reader,
	relativeArtifactPath string,
) error {
	uploadDescriptor, ok := uploader.uploadDescriptors[relativeArtifactPath]
	if !ok {
		return fmt.Errorf("no upload URL was generated for artifact path %s", relativeArtifactPath)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadDescriptor.url, artifact)
	if err != nil {
		return err
	}

	for key, value := range uploadDescriptor.headers {
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

	return nil
}

func (uploader *HTTPSUploader) Finish(ctx context.Context) error {
	paths := uploader.artifacts.UploadableRelativePaths()

	if len(paths) == 0 {
		return nil
	}

	_, err := client.CirrusClient.CommitUploadedArtifacts(ctx, &api.CommitUploadedArtifactsRequest{
		TaskIdentification: uploader.taskIdentification,
		Name:               uploader.artifacts.Name,
		Type:               uploader.artifacts.Type,
		Format:             uploader.artifacts.Format,
		Paths:              paths,
	})
	if err != nil {
		return err
	}

	return nil
}
