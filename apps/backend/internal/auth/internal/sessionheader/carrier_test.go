package sessionheader_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/sessionheader"
)

type handler struct {
	cookie    string
	setCookie []string
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.cookie = r.Header.Get("Cookie")
	for _, line := range h.setCookie {
		w.Header().Add("Set-Cookie", line)
	}
	_, _ = w.Write([]byte("{}"))
}

func serve(t *testing.T, next *handler, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth.v1.AuthService/CreateSession", nil)
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	recorder := httptest.NewRecorder()
	sessionheader.Carried(next).ServeHTTP(recorder, req)
	return recorder
}

func TestAPageWithTheCookieIsLeftAlone(t *testing.T) {
	next := &handler{setCookie: []string{"cp_sid=new; Path=/; HttpOnly; Secure; SameSite=Lax"}}

	res := serve(t, next, map[string]string{"Cookie": "cp_sid=old"})

	assert.Equal(t, "cp_sid=old", next.cookie)
	assert.Equal(t, []string{"cp_sid=new; Path=/; HttpOnly; Secure; SameSite=Lax"}, res.Header().Values("Set-Cookie"))
	assert.Empty(t, res.Header().Get(sessionheader.Name))
}

func TestTheCarriedSessionIsTheCookieTheHandlersRead(t *testing.T) {
	next := &handler{}

	serve(t, next, map[string]string{sessionheader.Name: "the-Token_1", "Cookie": "cp_sid=a-cookie-of-another-page"})

	assert.Equal(t, "cp_sid=the-Token_1", next.cookie)
}

func TestNoneOrAnythingButATokenIsNoCookie(t *testing.T) {
	for _, carried := range []string{sessionheader.None, "a; cp_oauth=sealed", "with space"} {
		t.Run(carried, func(t *testing.T) {
			next := &handler{}

			serve(t, next, map[string]string{sessionheader.Name: carried, "Cookie": "cp_sid=ignored"})

			assert.Empty(t, next.cookie)
		})
	}
}

func TestTheSessionSetIsAnsweredInTheHeaderAndNotAsACookie(t *testing.T) {
	next := &handler{setCookie: []string{
		"cp_sid=fresh; Path=/; Max-Age=2592000; HttpOnly; Secure; SameSite=Lax",
		"cp_oauth=; Path=/; Max-Age=0",
	}}

	res := serve(t, next, map[string]string{sessionheader.Name: sessionheader.None})

	assert.Equal(t, "fresh", res.Header().Get(sessionheader.Name))
	assert.Equal(t, []string{"cp_oauth=; Path=/; Max-Age=0"}, res.Header().Values("Set-Cookie"))
	assert.Equal(t, "{}", res.Body.String())
}

func TestAClearedSessionIsAnsweredNone(t *testing.T) {
	next := &handler{setCookie: []string{"cp_sid=; Path=/; Max-Age=0; HttpOnly; Secure; SameSite=Lax"}}

	res := serve(t, next, map[string]string{sessionheader.Name: "the-token"})

	assert.Equal(t, sessionheader.None, res.Header().Get(sessionheader.Name))
	assert.Empty(t, res.Header().Values("Set-Cookie"))
}

func TestAnAnswerThatSetsNoSessionCarriesNoHeader(t *testing.T) {
	res := serve(t, &handler{}, map[string]string{sessionheader.Name: "the-token"})

	require.Equal(t, http.StatusOK, res.Code)
	assert.Empty(t, res.Header().Values(sessionheader.Name), "the page keeps the token it holds")
}
