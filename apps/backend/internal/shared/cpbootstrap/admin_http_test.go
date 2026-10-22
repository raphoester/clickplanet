package cpbootstrap_test

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

func handlerModule(name, pattern string, dialed *string) cpbootstrap.Module {
	return newModule(name, func(props cpbootstrap.Props) error {
		if dialed != nil {
			_, base, err := props.Admin.Dial()
			if err != nil {
				return err //nolint:wrapcheck // the test reads the sentence.
			}
			*dialed = base
		}
		return props.AdminHTTP.Handle(pattern, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "served by "+name)
		}))
	})
}

func get(t *testing.T, url string) (int, http.Header, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Close = true
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	return res.StatusCode, res.Header, string(body)
}

func TestAHandlerIsServedOnTheAdminListenerAloneAndOutsideTheGatesMiddlewares(t *testing.T) {
	public, console, admin := listen(t), listen(t), listen(t)
	server := cpbootstrap.ServerConfig{
		BindAddress:        public.Addr().String(),
		ConsoleBindAddress: console.Addr().String(),
		AdminBindAddress:   admin.Addr().String(),
	}

	serveBehind(t, server, headerGate(), []net.Listener{public, console, admin}, func() {
		status, header, body := get(t, "http://"+server.AdminBindAddress+"/mcp")
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, "served by mcp", body)
		assert.Empty(t, header.Get(gateHeader), "the handler says who may call itself")

		status, _, _ = get(t, "http://"+server.ConsoleBindAddress+"/mcp")
		assert.Equal(t, http.StatusNotFound, status, "not on the console")
		status, _, _ = get(t, "http://"+server.BindAddress+"/mcp")
		assert.Equal(t, http.StatusNotFound, status, "not on the public router")

		require.NoError(t, call(t.Context(), server.ConsoleBindAddress), "the admin services are served as before")
	}, handlerModule("mcp", "/mcp", nil), adminModule(nil))
}

func TestAModuleDialsTheAdminListenerOnLoopback(t *testing.T) {
	public, admin := listen(t), listen(t)
	server := cpbootstrap.ServerConfig{BindAddress: public.Addr().String(), AdminBindAddress: admin.Addr().String()}
	var dialed string

	serveBehind(t, server, headerGate(), []net.Listener{public, admin}, func() {
		status, _, body := get(t, dialed+"/mcp")
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, "served by mcp", body)
	}, handlerModule("mcp", "/mcp", &dialed))

	assert.Equal(t, "http://"+server.AdminBindAddress, dialed)
}

func TestAnUnspecifiedAdminHostIsDialedOnLoopback(t *testing.T) {
	for address, base := range map[string]string{
		"0.0.0.0:8083": "http://127.0.0.1:8083",
		":8083":        "http://127.0.0.1:8083",
		"[::]:8083":    "http://127.0.0.1:8083",
		"backend:8083": "http://backend:8083",
	} {
		var dialed string
		public := listen(t)

		// The modules are built before anything listens, so the boot fails only after the dial.
		err := cpbootstrap.RunOn(t.Context(), cpbootstrap.Options{
			Server:  cpbootstrap.ServerConfig{BindAddress: public.Addr().String(), AdminBindAddress: address},
			Logger:  slog.New(slog.DiscardHandler),
			Modules: []cpbootstrap.Module{handlerModule("mcp", "/mcp", &dialed)},
		}, public)

		require.ErrorContains(t, err, "nothing gates it", address)
		assert.Equal(t, base, dialed, address)
	}
}

func TestTwoModulesClaimingOnePatternRefuseTheBoot(t *testing.T) {
	public, admin := listen(t), listen(t)

	err := cpbootstrap.RunOn(t.Context(), cpbootstrap.Options{
		Server:    cpbootstrap.ServerConfig{BindAddress: public.Addr().String(), AdminBindAddress: admin.Addr().String()},
		Logger:    slog.New(slog.DiscardHandler),
		Modules:   []cpbootstrap.Module{handlerModule("auth", "/oauth/token", nil), handlerModule("mcp", "/oauth/token", nil)},
		AdminGate: headerGate(),
	}, public, admin)

	require.ErrorContains(t, err, "which auth already serves")
}

func TestAPatternThatConflictsWithAnAdminServiceRefusesTheBoot(t *testing.T) {
	public, admin := listen(t), listen(t)

	err := cpbootstrap.RunOn(t.Context(), cpbootstrap.Options{
		Server:    cpbootstrap.ServerConfig{BindAddress: public.Addr().String(), AdminBindAddress: admin.Addr().String()},
		Logger:    slog.New(slog.DiscardHandler),
		Modules:   []cpbootstrap.Module{adminModule(nil), handlerModule("mcp", adminProcedure, nil)},
		AdminGate: headerGate(),
	}, public, admin)

	require.ErrorContains(t, err, "cannot serve")
}

func TestWithNoAdminListenerAModuleCannotDialIt(t *testing.T) {
	public := listen(t)

	err := cpbootstrap.RunOn(t.Context(), cpbootstrap.Options{
		Server:  cpbootstrap.ServerConfig{BindAddress: public.Addr().String()},
		Logger:  slog.New(slog.DiscardHandler),
		Modules: []cpbootstrap.Module{handlerModule("mcp", "/mcp", new(string))},
	}, public)

	require.ErrorContains(t, err, "adminBindAddress is empty")
}
