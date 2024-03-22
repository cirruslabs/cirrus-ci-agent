package ghacache

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-chi/render"
	"github.com/puzpuzpuz/xsync/v3"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const APIMountPoint = "/_apis/artifactcache"

type GHACache struct {
	cacheHost   string
	mux         *http.ServeMux
	uploadables *xsync.MapOf[int64, *uploadable]
}

type uploadable struct {
	Key     string
	Version string
	Buf     *bytes.Buffer
}

func New(cacheHost string) *GHACache {
	cache := &GHACache{
		cacheHost:   cacheHost,
		mux:         http.NewServeMux(),
		uploadables: xsync.NewMapOf[int64, *uploadable](),
	}

	cache.mux.HandleFunc("GET /cache", cache.get)
	cache.mux.HandleFunc("POST /caches", cache.reserveUploadable)
	cache.mux.HandleFunc("PATCH /caches/{id}", cache.updateUploadable)
	cache.mux.HandleFunc("POST /caches/{id}", cache.commitUploadable)

	return cache
}

func (cache *GHACache) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	cache.mux.ServeHTTP(writer, request)
}

func (cache *GHACache) get(writer http.ResponseWriter, request *http.Request) {
	keys := strings.Split(request.URL.Query().Get("keys"), ",")
	version := request.URL.Query().Get("version")

	// The first key is used for exact matching which we support
	httpCacheURL := cache.httpCacheURL(keys[0], version)

	resp, err := http.Head(httpCacheURL)
	if err != nil {
		log.Printf("GHA cache failed to retrieve %q: %v\n", httpCacheURL, err)
		writer.WriteHeader(http.StatusInternalServerError)

		return
	}

	if resp.StatusCode == http.StatusOK {
		jsonResp := struct {
			Key string `json:"cacheKey"`
			URL string `json:"archiveLocation"`
		}{
			Key: keys[0],
			URL: httpCacheURL,
		}

		render.JSON(writer, request, &jsonResp)

		return
	}

	// The rest of the keys are used for prefix matching
	// (fallback mechanism) which we do not support
	if len(keys[1:]) != 0 {
		log.Printf("GHA cache does not support prefix matching, was needed for (%v, %v)\n",
			keys, version)
		writer.WriteHeader(http.StatusBadRequest)

		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (cache *GHACache) reserveUploadable(writer http.ResponseWriter, request *http.Request) {
	var jsonReq struct {
		Key     string `json:"key"`
		Version string `json:"version"`
	}

	if err := render.DecodeJSON(request.Body, &jsonReq); err != nil {
		log.Printf("GHA cache failed to read/decode the JSON passed to the "+
			"reserve uploadable endpoint: %v\n", err)
		writer.WriteHeader(http.StatusBadRequest)

		return
	}

	jsonResp := struct {
		CacheID int64 `json:"cacheId"`
	}{
		CacheID: rand.Int63(),
	}

	cache.uploadables.Store(jsonResp.CacheID, &uploadable{
		Key:     jsonReq.Key,
		Version: jsonReq.Version,
		Buf:     &bytes.Buffer{},
	})

	render.JSON(writer, request, &jsonResp)
}

func (cache *GHACache) updateUploadable(writer http.ResponseWriter, request *http.Request) {
	id, ok := getID(request)
	if !ok {
		log.Printf("GHA cache failed to get/decode the ID passed to the " +
			"update uploadable endpoint\n")
		writer.WriteHeader(http.StatusBadRequest)

		return
	}

	uploadable, ok := cache.uploadables.Load(id)
	if !ok {
		writer.WriteHeader(http.StatusNotFound)

		return
	}

	if _, err := io.Copy(uploadable.Buf, request.Body); err != nil {
		if errors.Is(err, io.EOF) {
			return
		}

		writer.WriteHeader(http.StatusInternalServerError)
	}
}

func (cache *GHACache) commitUploadable(writer http.ResponseWriter, request *http.Request) {
	id, ok := getID(request)
	if !ok {
		log.Printf("GHA cache failed to get/decode the ID passed to the " +
			"commit uploadable endpoint\n")
		writer.WriteHeader(http.StatusBadRequest)

		return
	}

	uploadable, ok := cache.uploadables.Load(id)
	if !ok {
		writer.WriteHeader(http.StatusNotFound)

		return
	}
	defer cache.uploadables.Delete(id)

	var jsonReq struct {
		Size int64 `json:"size"`
	}

	if err := render.DecodeJSON(request.Body, &jsonReq); err != nil {
		log.Printf("GHA cache failed to read/decode the JSON passed to the "+
			"commit uploadable endpoint: %v\n", err)
		writer.WriteHeader(http.StatusBadRequest)

		return
	}

	if jsonReq.Size != int64(uploadable.Buf.Len()) {
		log.Printf("GHA cache detected a cache entry size mismatch for uploadable "+
			"with ID %d\n", id)
		writer.WriteHeader(http.StatusBadRequest)

		return
	}

	resp, err := http.Post(
		cache.httpCacheURL(uploadable.Key, uploadable.Version),
		"application/octet-stream",
		bytes.NewReader(uploadable.Buf.Bytes()),
	)
	if err != nil {
		log.Printf("GHA cache failed to upload the uploadable with ID %d: %v\n", id, err)
		writer.WriteHeader(http.StatusInternalServerError)

		return
	}

	if resp.StatusCode != http.StatusCreated {
		log.Printf("GHA cache failed to upload the uploadable with ID %d: got HTTP %d\n",
			id, resp.StatusCode)
		writer.WriteHeader(http.StatusInternalServerError)

		return
	}
}

func (cache *GHACache) httpCacheURL(key string, version string) string {
	return fmt.Sprintf("http://%s/%s-%s", cache.cacheHost, url.PathEscape(key), url.PathEscape(version))
}

func getID(request *http.Request) (int64, bool) {
	idRaw := request.PathValue("id")

	id, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil {
		return 0, false
	}

	return id, true
}
