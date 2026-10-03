package e2e_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconfigs"
)

func startSeasons(t *testing.T) string {
	t.Helper()

	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t), AllowedOrigin: "https://clickplanet.lol"}
	path := filepath.Join(t.TempDir(), "seasons.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
seasons:
  list:
    - number: 0
      endsAt: 2026-10-31T23:00:00Z
      finale: 2h
`), 0o600))
	var config struct{ Seasons seasons.Config }
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(path)))

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.Run(ctx, cpbootstrap.Options{
			Server:  server,
			Logger:  slog.New(slog.DiscardHandler),
			Modules: []cpbootstrap.Module{seasons.NewModule(config.Seasons)},
		})
	}()
	t.Cleanup(func() {
		cancel()
		assert.NoError(t, <-done)
	})

	require.Eventually(t, func() bool {
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", server.BindAddress)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 5*time.Second, 10*time.Millisecond, "the server never came up")

	return "http://" + server.BindAddress
}

func getFile(t *testing.T, url string) (int, http.Header, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	return res.StatusCode, res.Header, string(body)
}

func TestTheFinaleIsACalendarFileAtTheRouteTheProtoDeclares(t *testing.T) {
	status, header, body := getFile(t, startSeasons(t)+"/seasons/0/finale.ics")

	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "text/calendar; charset=utf-8", header.Get("Content-Type"))
	assert.Equal(t, `inline; filename="clickplanet-season-0-finale.ics"`, header.Get("Content-Disposition"))
	assert.Equal(t, "public, max-age=60", header.Get("Cache-Control"))
	assert.Contains(t, body, "\r\nDTSTART:20261031T210000Z\r\nDTEND:20261031T230000Z\r\n")
	assert.Contains(t, body, "\r\nURL:https://clickplanet.lol/play\r\n")
}

func TestAFinaleNotInTheCalendarIsNotFound(t *testing.T) {
	baseURL := startSeasons(t)

	status, _, _ := getFile(t, baseURL+"/seasons/1/finale.ics")
	assert.Equal(t, http.StatusNotFound, status)

	status, _, _ = getFile(t, baseURL+"/seasons/zero/finale.ics")
	assert.Equal(t, http.StatusBadRequest, status)
}
