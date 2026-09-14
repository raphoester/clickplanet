package admin_server_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/admin_server"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country"
)

type stubReassign struct {
	got reassign_country.In
	out reassign_country.Out
	err error
}

func (s *stubReassign) Execute(_ context.Context, in reassign_country.In) (reassign_country.Out, error) {
	s.got = in
	return s.out, s.err
}

func post(t *testing.T, handler http.Handler, body string) (int, map[string]any) {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, admin_server.ReassignPath, strings.NewReader(body))
	handler.ServeHTTP(rec, req)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &decoded), rec.Body.String())

	return rec.Code, decoded
}

func TestAReassignmentAnswersWhatItDid(t *testing.T) {
	useCase := &stubReassign{out: reassign_country.Out{FromBefore: 22040, ToBefore: 0, Moved: 22040, FromAfter: 0, ToAfter: 22040}}

	status, body := post(t, admin_server.NewHandler(useCase, nil), `{"from":"dz","to":"fr"}`)

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, reassign_country.In{From: "dz", To: "fr"}, useCase.got)
	assert.InDelta(t, 22040, body["moved"], 0)
	assert.InDelta(t, 22040, body["fromBefore"], 0)
	assert.InDelta(t, 22040, body["toAfter"], 0)
	assert.NotContains(t, body, "error")
}

func TestADryRunIsPassedThrough(t *testing.T) {
	useCase := &stubReassign{}

	status, body := post(t, admin_server.NewHandler(useCase, nil), `{"from":"dz","to":"fr","dryRun":true}`)

	assert.Equal(t, http.StatusOK, status)
	assert.True(t, useCase.got.DryRun)
	assert.Equal(t, true, body["dryRun"])
}

func TestACallerMistakeIsABadRequest(t *testing.T) {
	for name, err := range map[string]error{
		"unknown country": fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, "xx"),
		"same country":    reassign_country.ErrSameCountry,
	} {
		t.Run(name, func(t *testing.T) {
			status, body := post(t, admin_server.NewHandler(&stubReassign{err: err}, nil), `{"from":"xx","to":"xx"}`)

			assert.Equal(t, http.StatusBadRequest, status)
			assert.NotEmpty(t, body["error"])
		})
	}
}

func TestAMalformedBodyIsABadRequest(t *testing.T) {
	useCase := &stubReassign{}

	for _, body := range []string{`not json`, `{"from":"dz","to":"fr","everything":true}`} {
		status, _ := post(t, admin_server.NewHandler(useCase, nil), body)
		assert.Equal(t, http.StatusBadRequest, status, body)
	}
	assert.Empty(t, useCase.got.From, "nothing reached the use case")
}

func TestAFailureHalfwayStillSaysHowFarItGot(t *testing.T) {
	useCase := &stubReassign{
		out: reassign_country.Out{FromBefore: 10, Moved: 4, FromAfter: 6, ToAfter: 4},
		err: errors.New("reassignment interrupted: context canceled"),
	}

	status, body := post(t, admin_server.NewHandler(useCase, nil), `{"from":"dz","to":"fr"}`)

	assert.Equal(t, http.StatusInternalServerError, status)
	assert.InDelta(t, 4, body["moved"], 0)
	assert.InDelta(t, 6, body["fromAfter"], 0)
}

func TestOnlyPostIsServed(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, admin_server.ReassignPath, nil)
	admin_server.NewHandler(&stubReassign{}, nil).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestOnlyALoopbackAddressIsAccepted(t *testing.T) {
	for addr, ok := range map[string]bool{
		"":               true, // the default, 127.0.0.1:8081
		"127.0.0.1:9000": true,
		"localhost:8081": true,
		"[::1]:8081":     true,
		":8081":          false,
		"0.0.0.0:8081":   false,
		"10.0.0.5:8081":  false,
		"[::]:8081":      false,
		"127.0.0.1":      false,
	} {
		err := admin_server.Config{Enabled: true, BindAddress: addr}.Validate()
		if ok {
			require.NoError(t, err, addr)
		} else {
			require.Error(t, err, addr)
		}
	}

	require.NoError(t, admin_server.Config{Enabled: false, BindAddress: "0.0.0.0:8081"}.Validate(),
		"an address nothing listens on is not checked")
}

func TestItServesOnItsListenerUntilTheContextEnds(t *testing.T) {
	listener, err := admin_server.Listen(admin_server.Config{Enabled: true, BindAddress: "127.0.0.1:0"})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		admin_server.Serve(ctx, listener, admin_server.NewHandler(&stubReassign{}, nil), nil)
	}()

	url := "http://" + listener.Addr().String() + admin_server.ReassignPath
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, strings.NewReader(`{"from":"dz","to":"fr","dryRun":true}`))
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	cancel()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("the admin server did not stop")
	}
}

func TestListenRefusesAnAddressOutsideTheContainer(t *testing.T) {
	_, err := admin_server.Listen(admin_server.Config{Enabled: true, BindAddress: ":0"})
	require.Error(t, err)
}
