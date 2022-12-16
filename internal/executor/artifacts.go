package executor

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/bmatcuk/doublestar"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"github.com/cirruslabs/cirrus-ci-agent/internal/environment"
	"github.com/cirruslabs/cirrus-ci-annotations"
	"github.com/cirruslabs/cirrus-ci-annotations/model"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"os"
	"path/filepath"
)

type ArtifactUploader interface {
	Upload(ctx context.Context, artifact io.Reader, relativeArtifactPath string) error
	Finish(ctx context.Context) error
}

type InstantiateArtifactUploaderFunc func(
	ctx context.Context,
	taskIdentification *api.TaskIdentification,
	artifactName string,
	artifactType string,
	artifactFormat string,
) (ArtifactUploader, error)

type ProcessedPath struct {
	Pattern string
	Paths   []string
}

var ErrArtifactsPathOutsideWorkingDir = errors.New("path is outside of CIRRUS_WORKING_DIR")

func (executor *Executor) UploadArtifacts(
	ctx context.Context,
	logUploader *LogUploader,
	name string,
	artifactsInstruction *api.ArtifactsInstruction,
	customEnv *environment.Environment,
) bool {
	// Check if we need to upload anything at all
	if len(artifactsInstruction.Paths) == 0 {
		fmt.Fprintln(logUploader, "Skipping artifacts upload because there are no paths specified...")

		return true
	}

	// Upload artifacts: try first via HTTPS, then fallback via gRPC
	uploadedPaths, success := executor.uploadArtifactsWithRetries(ctx, "HTTPS", NewHTTPSUploader, name,
		logUploader, artifactsInstruction, customEnv)
	if !success {
		uploadedPaths, success = executor.uploadArtifactsWithRetries(ctx, "gRPC", NewGRPCUploader, name,
			logUploader, artifactsInstruction, customEnv)
		if !success {
			return false
		}
	}

	// Process and upload annotations
	if artifactsInstruction.Format != "" {
		return executor.processAndUploadAnnotations(ctx, customEnv.Get("CIRRUS_WORKING_DIR"),
			uploadedPaths, logUploader, artifactsInstruction.Format)
	}

	return true
}

func (executor *Executor) uploadArtifactsWithRetries(
	ctx context.Context,
	method string,
	instantiateArtifactUploader InstantiateArtifactUploaderFunc,
	name string,
	logUploader *LogUploader,
	artifactsInstruction *api.ArtifactsInstruction,
	customEnv *environment.Environment,
) (result []string, success bool) {
	fmt.Fprintf(logUploader, "Trying to upload artifacts over %s...\n", method)

	artifactUploader, err := instantiateArtifactUploader(ctx, executor.taskIdentification, name,
		artifactsInstruction.Type, artifactsInstruction.Format)
	if err != nil {
		fmt.Fprintf(logUploader, "Failed to initialize %s artifact uploader: %v\n", method, err)

		return nil, false
	}
	defer func() {
		if err := artifactUploader.Finish(ctx); err != nil {
			fmt.Fprintf(logUploader, "Failed to finalize %s artifact uploader: %v\n", method, err)
			success = false
		}
	}()

	err = retry.Do(
		func() error {
			result, err = executor.uploadArtifacts(ctx, artifactsInstruction, customEnv, logUploader, artifactUploader)
			return err
		}, retry.OnRetry(func(n uint, err error) {
			fmt.Fprintf(logUploader, "Failed to upload artifacts: %v\n", err)
			fmt.Fprintln(logUploader, "Re-trying to artifacts upload...")
		}),
		retry.Attempts(2),
		retry.Context(ctx),
		retry.RetryIf(func(err error) bool {
			if errors.Is(err, ErrArtifactsPathOutsideWorkingDir) {
				return false
			}

			if status, ok := status.FromError(err); ok {
				if status.Code() == codes.Unimplemented {
					return false
				}
			}

			return true
		}),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		if errors.Is(err, ErrArtifactsPathOutsideWorkingDir) {
			fmt.Fprintf(logUploader, "Failed to upload artifacts: %v\n", err)

			return nil, false
		}

		if status, ok := status.FromError(err); ok {
			if status.Code() == codes.Unimplemented {
				fmt.Fprintf(logUploader, "Artifact upload over %s is not supported\n", method)

				return nil, false
			}
		}

		fmt.Fprintf(logUploader, "Failed to upload artifacts after multiple tries: %s\n", err)

		return nil, false
	}

	return result, true
}

func (executor *Executor) uploadArtifacts(
	ctx context.Context,
	artifactsInstruction *api.ArtifactsInstruction,
	customEnv *environment.Environment,
	logUploader *LogUploader,
	artifactUploader ArtifactUploader,
) ([]string, error) {
	var result []string

	workingDir := customEnv.Get("CIRRUS_WORKING_DIR")

	var processedPaths []ProcessedPath

	for _, path := range artifactsInstruction.Paths {
		pattern := customEnv.ExpandText(path)
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(workingDir, pattern)
		}

		paths, err := doublestar.Glob(pattern)
		if err != nil {
			return result, errors.Wrap(err, "Failed to list artifacts")
		}

		// Ensure that the all resulting paths are scoped to the CIRRUS_WORKING_DIR
		for _, artifactPath := range paths {
			matcher := filepath.Join(workingDir, "**")
			matched, err := doublestar.PathMatch(matcher, artifactPath)
			if err != nil {
				return result, errors.Wrapf(err, "failed to match the path: %v", err)
			}
			if !matched {
				return result, fmt.Errorf("%w: path %s should be relative to %s",
					ErrArtifactsPathOutsideWorkingDir, artifactPath, workingDir)
			}
		}

		processedPaths = append(processedPaths, ProcessedPath{Pattern: pattern, Paths: paths})
	}

	for _, processedPath := range processedPaths {
		fmt.Fprintf(logUploader, "Uploading %d artifacts for %s\n", len(processedPath.Paths),
			processedPath.Pattern)

		for _, artifactPath := range processedPath.Paths {
			info, err := os.Stat(artifactPath)

			if err == nil && info.IsDir() {
				fmt.Fprintf(logUploader, "Skipping uploading of '%s' because it's a folder\n", artifactPath)
				continue
			}

			if err == nil && info.Size() > 100*humanize.MByte {
				fmt.Fprintf(logUploader, "\nUploading a quite hefty artifact '%s' of size %s\n", artifactPath,
					humanize.Bytes(uint64(info.Size())))
			}

			relativeArtifactPath, err := filepath.Rel(workingDir, artifactPath)
			if err != nil {
				return result, errors.Wrapf(err, "failed to get artifact relative path for %s", artifactPath)
			}
			relativeArtifactPath = filepath.ToSlash(relativeArtifactPath)

			artifactFile, err := os.Open(artifactPath)
			if err != nil {
				return result, errors.Wrapf(err, "failed to read artifact file %s", artifactPath)
			}

			err = artifactUploader.Upload(ctx, artifactFile, relativeArtifactPath)
			if err != nil {
				_ = artifactFile.Close()
				return result, err
			}

			_ = artifactFile.Close()

			fmt.Fprintf(logUploader, "Uploaded %s\n", artifactPath)

			result = append(result, artifactPath)
		}
	}

	return result, nil
}

func (executor *Executor) processAndUploadAnnotations(
	ctx context.Context,
	workingDir string,
	uploadedPaths []string,
	logUploader *LogUploader,
	format string,
) bool {
	var allAnnotations []model.Annotation

	for _, uploadedPath := range uploadedPaths {
		fmt.Fprintf(logUploader, "Trying to parse annotations for %s format\n", format)

		err, artifactAnnotations := annotations.ParseAnnotations(format, uploadedPath)
		if err != nil {
			fmt.Fprintf(logUploader, "failed to create annotations from %s: %v", uploadedPath, err)

			return false
		}

		allAnnotations = append(allAnnotations, artifactAnnotations...)
	}

	if len(allAnnotations) == 0 {
		return true
	}

	normalizedAnnotations, err := annotations.NormalizeAnnotations(workingDir, allAnnotations)
	if err != nil {
		fmt.Fprintf(logUploader, "Failed to validate annotations: %v\n", err)
	}
	protoAnnotations := ConvertAnnotations(normalizedAnnotations)
	reportAnnotationsCommandRequest := api.ReportAnnotationsCommandRequest{
		TaskIdentification: executor.taskIdentification,
		Annotations:        protoAnnotations,
	}

	err = retry.Do(
		func() error {
			_, err = client.CirrusClient.ReportAnnotations(ctx, &reportAnnotationsCommandRequest)
			return err
		}, retry.OnRetry(func(n uint, err error) {
			fmt.Fprintf(logUploader, "\nFailed to report %d annotations: %s", len(normalizedAnnotations), err)
			fmt.Fprintln(logUploader, "Retrying...")
		}),
		retry.Attempts(2),
		retry.Context(ctx),
	)
	if err != nil {
		fmt.Fprintf(logUploader, "Still failed to report %d annotations: %s. Ignoring...\n",
			len(normalizedAnnotations), err)

		return true
	}

	fmt.Fprintf(logUploader, "Reported %d annotations!\n", len(normalizedAnnotations))

	return true
}
