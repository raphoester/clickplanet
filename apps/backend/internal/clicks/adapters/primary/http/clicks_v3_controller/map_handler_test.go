package clicks_v3_controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubEncoder struct{ body []byte }

func (s stubEncoder) EncodeStateBatch(start uint32, end uint32) ([]byte, error) {
	if start > end {
		return nil, fmt.Errorf("invalid range")
	}
	return s.body, nil
}

func newMapServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle("GET /map", NewMapHandler(stubEncoder{body: []byte("CPM1")}, stubChecker{}, nil))

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

func TestServeHTTP(t *testing.T) {
	get := func(t *testing.T, query string) *http.Response {
		t.Helper()

		server := newMapServer(t)
		res, err := server.Client().Get(server.URL + "/map" + query)
		require.NoError(t, err)
		t.Cleanup(func() { _ = res.Body.Close() })

		return res
	}

	t.Run("answers the encoded chunk, cacheable", func(t *testing.T) {
		res := get(t, "")
		require.Equal(t, http.StatusOK, res.StatusCode)
		require.Equal(t, "application/octet-stream", res.Header.Get("Content-Type"))
		require.Equal(t, "public, max-age=5", res.Header.Get("Cache-Control"))
	})

	t.Run("a range that is not a number is refused", func(t *testing.T) {
		require.Equal(t, http.StatusBadRequest, get(t, "?start=abc").StatusCode)
	})

	t.Run("an inverted range is refused", func(t *testing.T) {
		require.Equal(t, http.StatusBadRequest, get(t, "?start=9&end=2").StatusCode)
	})
}
