package http_cache

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/internal/client"
	"io"
	"log"
	"net/http"
)

func downloadCacheViaRPC(w http.ResponseWriter, r *http.Request, cacheKey string) {
	cacheStream, err := client.CirrusClient.DownloadCache(r.Context(), &api.DownloadCacheRequest{
		TaskIdentification: cirrusTaskIdentification,
		CacheKey:           cacheKey,
	})
	if err != nil {
		log.Printf("%s cache download initialization (RPC fallback) failed: %v\n", cacheKey, err)
		w.WriteHeader(http.StatusNotFound)

		return
	}

	for {
		chunk, err := cacheStream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Printf("%s cache download (RPC fallback) finished...\n", cacheKey)
			} else {
				log.Printf("%s cache download (RPC fallback) failed: %v\n", cacheKey, err)
				w.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		if chunk.RedirectUrl != "" {
			log.Printf("%s cache download (RPC fallback) requested a redirect\n", cacheKey)
			proxyDownloadFromURL(w, chunk.RedirectUrl)

			return
		}

		if len(chunk.Data) == 0 {
			continue
		}

		if _, err := w.Write(chunk.Data); err != nil {
			log.Printf("%s cache download (RPC fallback) failed: %v\n", cacheKey, err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}
}

func uploadCacheEntryViaRPC(w http.ResponseWriter, r *http.Request, cacheKey string) {
	uploadCacheClient, err := client.CirrusClient.UploadCache(r.Context())
	if err != nil {
		log.Printf("%s cache upload initialization (RPC fallback) failed: %v\n", cacheKey, err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if err := uploadCacheClient.Send(&api.CacheEntry{
		Value: &api.CacheEntry_Key{
			Key: &api.CacheKey{
				TaskIdentification: cirrusTaskIdentification,
				CacheKey:           cacheKey,
			},
		},
	}); err != nil {
		log.Printf("%s cache upload (RPC fallback) failed: %v\n", cacheKey, err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	buf := make([]byte, 1024*1024)

	for {
		n, err := r.Body.Read(buf)
		if err == io.EOF {
			log.Printf("%s cache upload (RPC fallback) finished...\n", cacheKey)

			break
		}
		if err != nil {
			log.Printf("%s cache upload (RPC fallback) failed: %v\n", cacheKey, err)
			w.WriteHeader(http.StatusBadRequest)

			_, _ = uploadCacheClient.CloseAndRecv()

			return
		}

		if n == 0 {
			continue
		}

		err = uploadCacheClient.Send(&api.CacheEntry{
			Value: &api.CacheEntry_Chunk{
				Chunk: &api.DataChunk{
					Data: buf[:n],
				},
			},
		})
		if err != nil {
			log.Printf("%s cache upload (RPC fallback) failed: %v\n", cacheKey, err)
			w.WriteHeader(http.StatusInternalServerError)

			_, _ = uploadCacheClient.CloseAndRecv()

			return
		}
	}

	if _, err := uploadCacheClient.CloseAndRecv(); err != nil {
		log.Printf("%s cache upload (RPC fallback) failed: %v\n", cacheKey, err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
}
