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

func TestTheFinaleIsACalendarFileLinkingBackToTheGame(t *testing.T) {
	baseURL := startSeasons(t)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, baseURL+"/seasons/0/finale.ics", nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "text/calendar; charset=utf-8", res.Header.Get("Content-Type"))
	assert.Contains(t, string(body), "\r\nDTSTART:20261031T210000Z\r\nDTEND:20261031T230000Z\r\n")
	assert.Contains(t, string(body), "\r\nURL:https://clickplanet.lol/play\r\n")
}
