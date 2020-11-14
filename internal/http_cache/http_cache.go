package http_cache

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/certifi/gocertifi"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"golang.org/x/sync/semaphore"
	"io"
	"log"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

var cirrusTaskIdentification *api.TaskIdentification

const (
	activeRequestsPerLogicalCPU = 4

	CirrusHeaderCreatedBy = "Cirrus-Created-By"
)

var sem = semaphore.NewWeighted(int64(runtime.NumCPU() * activeRequestsPerLogicalCPU))

var httpProxyClient = &http.Client{}

func Start(taskIdentification *api.TaskIdentification) string {
	cirrusTaskIdentification = taskIdentification

	certPool, err := gocertifi.CACerts()
	if err == nil {
		httpProxyClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: certPool},
			},
			Timeout: time.Minute,
		}
	}

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
	// Limit request concurrency
	if err := sem.Acquire(r.Context(), 1); err != nil {
		log.Printf("Failed to acquite the semaphore: %s\n", err)
		if errors.Is(err, context.Canceled) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			w.WriteHeader(http.StatusRequestTimeout)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func() {
		sem.Release(1)
	}()

	key := r.URL.Path
	if key[0] == '/' {
		key = key[1:]
	}
	if len(key) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if r.Method == "GET" {
		downloadCache(w, r, key)
	} else if r.Method == "HEAD" {
		checkCacheExists(w, key)
	} else if r.Method == "POST" {
		uploadCache(w, r, key)
	} else if r.Method == "PUT" {
		uploadCache(w, r, key)
	} else {
		log.Printf("Not supported request method: %s\n", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func checkCacheExists(w http.ResponseWriter, cacheKey string) {
	cacheInfoRequest := api.CacheInfoRequest{
		TaskIdentification: cirrusTaskIdentification,
		CacheKey:           cacheKey,
	}
	response, err := client.CirrusClient.CacheInfo(context.Background(), &cacheInfoRequest)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		if response.Info.CreatedByTaskId > 0 {
			w.Header().Set(CirrusHeaderCreatedBy, strconv.FormatInt(response.Info.CreatedByTaskId, 10))
		}
		w.Header().Set("Content-Length", strconv.FormatInt(response.Info.SizeInBytes, 10))
		w.WriteHeader(http.StatusOK)
	}
}

func downloadCache(w http.ResponseWriter, r *http.Request, cacheKey string) {
	downloadCacheRequest := api.DownloadCacheRequest{
		TaskIdentification: cirrusTaskIdentification,
		CacheKey:           cacheKey,
	}
	cacheStream, err := client.CirrusClient.DownloadCache(context.Background(), &downloadCacheRequest)
	if err != nil {
		log.Println("Not found!")
		w.WriteHeader(http.StatusNotFound)
	} else {
		for {
			in, err := cacheStream.Recv()
			if in != nil && in.RedirectUrl != "" {
				log.Printf("Redirecting cache download of %s\n", cacheKey)
				proxyDownloadFromURL(w, in.RedirectUrl)
				break
			}
			if in != nil && in.Data != nil && len(in.Data) > 0 {
				_, _ = w.Write(in.Data)
			}
			if err == io.EOF {
				w.WriteHeader(http.StatusOK)
				log.Printf("Finished downloading %s...\n", cacheKey)
				break
			}
			if err != nil {
				log.Printf("Failed to download %s cache! %s", cacheKey, err)
				w.WriteHeader(http.StatusNotFound)
				break
			}
		}
	}
}

func proxyDownloadFromURL(w http.ResponseWriter, url string) {
	resp, err := httpProxyClient.Get(url)
	if err != nil {
		log.Printf("Proxying cache %s failed: %v\n", url, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	successfulStatus := 100 <= resp.StatusCode && resp.StatusCode < 300
	if !successfulStatus {
		log.Printf("Proxying cache %s failed with %d status\n", url, resp.StatusCode)
		w.WriteHeader(resp.StatusCode)
		return
	}
	bytesRead, err := io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Proxying cache download for %s failed with %v\n", url, err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		log.Printf("Proxying cache %s succeded! Proxies %d bytes!\n", url, bytesRead)
		w.WriteHeader(http.StatusOK)
	}
}

func uploadCache(w http.ResponseWriter, r *http.Request, cacheKey string) {
	uploadCacheClient, err := client.CirrusClient.UploadCache(context.Background())
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to initialized uploading of %s cache! %s", cacheKey, err)
		log.Print(errorMsg)
		w.Write([]byte(errorMsg))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	cacheKeyMsg := api.CacheEntry_CacheKey{TaskIdentification: cirrusTaskIdentification, CacheKey: cacheKey}
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
				log.Print(errorMsg)
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
			log.Print(errorMsg)
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
