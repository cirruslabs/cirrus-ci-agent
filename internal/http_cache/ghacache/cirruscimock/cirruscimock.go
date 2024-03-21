package cirruscimock

import (
	"bytes"
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"io"
	"net"
	"testing"
)

type cirrusCIMock struct {
	uploadedCacheEntries map[string]*bytes.Buffer

	api.UnimplementedCirrusCIServiceServer
}

func ClientConn(t *testing.T) *grpc.ClientConn {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		server := grpc.NewServer()
		api.RegisterCirrusCIServiceServer(server, &cirrusCIMock{
			uploadedCacheEntries: map[string]*bytes.Buffer{},
		})
		require.NoError(t, server.Serve(lis))
	}()

	clientConn, err := grpc.DialContext(context.Background(), lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	return clientConn
}

func (mock *cirrusCIMock) UploadCache(stream api.CirrusCIService_UploadCacheServer) error {
	var currentBuf *bytes.Buffer

	for {
		cacheEntry, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return stream.SendAndClose(&api.UploadCacheResponse{})
			}

			return err
		}

		switch typed := cacheEntry.Value.(type) {
		case *api.CacheEntry_Key:
			currentBuf = &bytes.Buffer{}
			mock.uploadedCacheEntries[typed.Key.CacheKey] = currentBuf
		case *api.CacheEntry_Chunk:
			currentBuf.Write(typed.Chunk.Data)
		}
	}
}

func (mock *cirrusCIMock) DownloadCache(request *api.DownloadCacheRequest, stream api.CirrusCIService_DownloadCacheServer) error {
	buf, ok := mock.uploadedCacheEntries[request.CacheKey]
	if !ok {
		return status.Errorf(codes.NotFound, "cache entry for key %s is not found",
			request.CacheKey)
	}

	return stream.Send(&api.DataChunk{Data: buf.Bytes()})
}

func (mock *cirrusCIMock) CacheInfo(ctx context.Context, request *api.CacheInfoRequest) (*api.CacheInfoResponse, error) {
	buf, ok := mock.uploadedCacheEntries[request.CacheKey]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "cache entry for key %s is not found",
			request.CacheKey)
	}

	return &api.CacheInfoResponse{
		Info: &api.CacheInfo{
			Key:         request.CacheKey,
			SizeInBytes: int64(buf.Len()),
		},
	}, nil
}
