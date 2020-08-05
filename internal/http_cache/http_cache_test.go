package http_cache_test

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"github.com/cirruslabs/cirrus-ci-agent/internal/http_cache"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestHighLoad ensures that the HTTP cache can handle multiple concurrent connections.
func TestHighLoad(t *testing.T) {
	// Start a dummy gRPC server
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()
	api.RegisterCirrusCIServiceServer(server, &api.UnimplementedCirrusCIServiceServer{})

	go func() {
		if err := server.Serve(listener); err != nil {
			if !errors.Is(err, grpc.ErrServerStopped) {
				panic(err)
			}
		}
	}()

	// Initialize agent's gRPC client
	conn, err := grpc.Dial(listener.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	client.InitClient(conn)

	// Start HTTP cache
	cacheAddr := http_cache.Start(api.TaskIdentification{})

	// Wait for the HTTP cache to start
	_, err = http.Head(fmt.Sprintf("http://%s/key-doesnt-matter", cacheAddr))
	if err != nil {
		t.Fatal(err)
	}

	// Start 500 goroutines and ensure that they all get what they were looking for
	var wg sync.WaitGroup
	var successes uint64
	const numGoroutines = 500
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			cli := http.Client{
				Timeout: time.Second * 30,
			}
			resp, err := cli.Head(fmt.Sprintf("http://%s/key-doesnt-matter", cacheAddr))
			if err != nil {
				return
			}

			if resp.StatusCode == 404 {
				atomic.AddUint64(&successes, 1)
			}
		}()
	}

	wg.Wait()
	server.GracefulStop()
	assert.Equal(t, numGoroutines, atomic.LoadUint64(&successes))
}
