package http_cache

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"io"
	"log"
	"net"
	"net/http"
	"runtime"
)

var cirrusTaskIdentification api.TaskIdentification

const activeRequestsPerLogicalCPU = 4

var semaphore = make(chan struct{}, runtime.NumCPU()*activeRequestsPerLogicalCPU)

func Start(taskIdentification api.TaskIdentification) string {
	cirrusTaskIdentification = taskIdentification
	http.HandleFunc("/", handler)

	address := "127.0.0.1:12321"
	listener, err := net.Listen("tcp", address)

	if err != nil {
		log.Printf("Port 12321 is occupied: %s. Looking for another one...\n", err)
		listener, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err == nil {
		address = listener.Addr().String()
		log.Printf("Starting http cache server %s\n", address)
		go http.Serve(listener, nil)
	} else {
		log.Printf("Failed to start http cache server %s: %s\n", address, err)
	}
	return address
}

func handler(w http.ResponseWriter, r *http.Request) {
	// Grab semaphore slot
	select {
	case semaphore <- struct{}{}:
		break
	case <-r.Context().Done():
		if errors.Is(r.Context().Err(), context.Canceled) {
			return
		}
		if errors.Is(r.Context().Err(), context.DeadlineExceeded) {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		log.Printf("Failed to wait for the semaphore slot: %s\n", r.Context().Err())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Schedule the release of the semaphore slot
	defer func() {
		<-semaphore
	}()

	key := r.URL.Path
	if key[0] == '/' {
		key = key[1:]
	}
	if r.Method == "GET" {
		downloadCache(w, key)
	} else if r.Method == "HEAD" {
		checkCacheExists(w, key)
	} else if r.Method == "POST" {
		uploadCache(w, r, key)
	} else if r.Method == "PUT" {
		uploadCache(w, r, key)
	}
}

func checkCacheExists(w http.ResponseWriter, cacheKey string) {
	cacheInfoRequest := api.CacheInfoRequest{
		TaskIdentification: &cirrusTaskIdentification,
		CacheKey:           cacheKey,
	}
	_, err := client.CirrusClient.CacheInfo(context.Background(), &cacheInfoRequest)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func downloadCache(w http.ResponseWriter, cacheKey string) {
	downloadCacheRequest := api.DownloadCacheRequest{
		TaskIdentification: &cirrusTaskIdentification,
		CacheKey:           cacheKey,
	}
	cacheStream, err := client.CirrusClient.DownloadCache(context.Background(), &downloadCacheRequest)
	if err != nil {
		log.Print("Not found!")
		w.WriteHeader(http.StatusNotFound)
	} else {
		for {
			in, err := cacheStream.Recv()
			if in != nil && in.Data != nil && len(in.Data) > 0 {
				_, _ = w.Write(in.Data)
			}
			if err == io.EOF {
				w.WriteHeader(http.StatusOK)
				log.Printf("Finished downloading %s...\n", cacheKey)
				break
			}
			if err != nil {
				errorMsg := fmt.Sprintf("Failed to download %s cache! %s", cacheKey, err)
				log.Printf(errorMsg)
				w.WriteHeader(http.StatusNotFound)
				break
			}
		}
	}
}

func uploadCache(w http.ResponseWriter, r *http.Request, cacheKey string) {
	uploadCacheClient, err := client.CirrusClient.UploadCache(context.Background())
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to initialized uploading of %s cache! %s", cacheKey, err)
		log.Printf(errorMsg)
		w.Write([]byte(errorMsg))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	cacheKeyMsg := api.CacheEntry_CacheKey{TaskIdentification: &cirrusTaskIdentification, CacheKey: cacheKey}
	keyMsg := api.CacheEntry_Key{Key: &cacheKeyMsg}
	uploadCacheClient.Send(&api.CacheEntry{Value: &keyMsg})

	readBufferSize := int(1024 * 1024)
	readBuffer := make([]byte, readBufferSize)
	bufferedBodyReader := bufio.NewReaderSize(r.Body, readBufferSize)
	bytesUploaded := 0
	for {
		n, err := bufferedBodyReader.Read(readBuffer)

		if n > 0 {
			chunkMsg := api.CacheEntry_Chunk{Chunk: &api.DataChunk{Data: readBuffer[:n]}}
			err := uploadCacheClient.Send(&api.CacheEntry{Value: &chunkMsg})
			if err != nil {
				errorMsg := fmt.Sprintf("Failed to send a chunk: %s!", err)
				log.Printf(errorMsg)
				w.Write([]byte(errorMsg))
				w.WriteHeader(http.StatusInternalServerError)
				uploadCacheClient.CloseAndRecv()
				break
			}
			bytesUploaded += n
		}

		if err == io.EOF || n == 0 {
			uploadCacheClient.CloseAndRecv()
			w.WriteHeader(http.StatusCreated)
			break
		}
		if err != nil {
			errorMsg := fmt.Sprintf("Failed read cache body! %s", err)
			log.Printf(errorMsg)
			w.Write([]byte(errorMsg))
			w.WriteHeader(http.StatusBadRequest)
			uploadCacheClient.CloseAndRecv()
			break
		}
	}
	if bytesUploaded < 1024 {
		w.Write([]byte(fmt.Sprintf("Uploaded %d bytes.\n", bytesUploaded)))
	} else if bytesUploaded < 1024*1024 {
		w.Write([]byte(fmt.Sprintf("Uploaded %dKb.\n", bytesUploaded/1024)))
	} else {
		w.Write([]byte(fmt.Sprintf("Uploaded %dMb.\n", bytesUploaded/1024/1024)))
	}
}
