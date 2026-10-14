package e2e_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
)

type storedFile struct {
	contentType string
	body        []byte
}

type fakeBucket struct {
	url string

	mu    sync.Mutex
	files map[string]storedFile
}

func startBucket(t *testing.T) *fakeBucket {
	t.Helper()

	bucket := &fakeBucket{files: map[string]storedFile{}}
	server := httptest.NewServer(http.HandlerFunc(bucket.serve))
	t.Cleanup(server.Close)
	bucket.url = server.URL
	return bucket
}

func (b *fakeBucket) serve(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if r.URL.Query().Get("X-Amz-Signature") == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		b.files[r.URL.Path] = storedFile{contentType: r.Header.Get("Content-Type"), body: body}
	case http.MethodGet, http.MethodHead:
		file, kept := b.files[r.URL.Path]
		if !kept {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", file.contentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(file.body)))
		if r.Method == http.MethodGet {
			_, _ = w.Write(file.body)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
