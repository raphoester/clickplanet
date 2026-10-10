package httpapi_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-ops/internal/access"
	"github.com/raphoester/clickplanet.lol-ops/internal/accesslog"
	"github.com/raphoester/clickplanet.lol-ops/internal/excerpt"
	"github.com/raphoester/clickplanet.lol-ops/internal/httpapi"
	"github.com/raphoester/clickplanet.lol-ops/internal/journal"
	"github.com/raphoester/clickplanet.lol-ops/internal/sqlquery"
)

const assertion = "a-signed-assertion"

type desk struct {
	statements *sqlquery.StubExecutor
	journals   *journal.StubReader
	accessLogs *accesslog.StubReader
	handler    http.Handler
}

func newDesk() desk {
	statements := &sqlquery.StubExecutor{}
	journals := &journal.StubReader{}
	accessLogs := &accesslog.StubReader{}
	return desk{
		statements: statements,
		journals:   journals,
		accessLogs: accessLogs,
		handler: httpapi.NewHandler(
			access.StaticVerifier{assertion: "claude-cloud.access"},
			statements, journals, accessLogs,
		),
	}
}

func (d desk) ask(t *testing.T, method, target, body string) (int, http.Header, string) {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	request.Header.Set("Cf-Access-Jwt-Assertion", assertion)
	return d.serve(t, request)
}

func (d desk) serve(t *testing.T, request *http.Request) (int, http.Header, string) {
	t.Helper()

	recorder := httptest.NewRecorder()
	d.handler.ServeHTTP(recorder, request)
	answer, err := io.ReadAll(recorder.Result().Body)
	require.NoError(t, err)
	return recorder.Code, recorder.Result().Header, string(answer)
}

func TestACallerWithoutAnAssertionReachesNothing(t *testing.T) {
	for _, route := range []struct{ method, target string }{
		{http.MethodPost, "/sql"},
		{http.MethodGet, "/journal/cp-backend?since=2026-10-08T08:00:00Z"},
		{http.MethodGet, "/caddy?since=2026-10-08T08:00:00Z"},
	} {
		desk := newDesk()
		request := httptest.NewRequestWithContext(t.Context(), route.method, route.target, strings.NewReader("SELECT 1"))

		status, _, _ := desk.serve(t, request)

		assert.Equal(t, http.StatusUnauthorized, status, route.target)
		assert.Empty(t, desk.statements.Asked)
		assert.Empty(t, desk.journals.Asked)
		assert.Empty(t, desk.accessLogs.Asked)
	}
}

func TestACallerWithAnAssertionNobodySignedReachesNothing(t *testing.T) {
	desk := newDesk()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/sql", strings.NewReader("SELECT 1"))
	request.Header.Set("Cf-Access-Jwt-Assertion", "forged")

	status, _, _ := desk.serve(t, request)

	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Empty(t, desk.statements.Asked)
}

func TestAStatementAnswersWithItsRows(t *testing.T) {
	desk := newDesk()
	desk.statements.Result = sqlquery.Result{
		Columns: []string{"kind", "count"},
		Rows:    [][]any{{"take", int64(12)}, {"bomb", nil}},
	}

	status, header, answer := desk.ask(t, http.MethodPost, "/sql", "\nSELECT kind, count(*) FROM planet.ledger_takes GROUP BY kind\n")

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "application/json", header.Get("Content-Type"))
	assert.JSONEq(t, `{"columns":["kind","count"],"rows":[["take",12],["bomb",null]],"truncated":false}`, answer)
	assert.Equal(t, []sqlquery.Query{{
		Statement: "SELECT kind, count(*) FROM planet.ledger_takes GROUP BY kind",
		MaxRows:   1000,
		MaxBytes:  8 << 20,
	}}, desk.statements.Asked)
}

func TestAStatementCanAskForMoreRows(t *testing.T) {
	desk := newDesk()

	status, _, _ := desk.ask(t, http.MethodPost, "/sql?limit=20000", "SELECT 1")

	assert.Equal(t, http.StatusOK, status)
	require.Len(t, desk.statements.Asked, 1)
	assert.Equal(t, 20000, desk.statements.Asked[0].MaxRows)
}

func TestAStatementPostgresRefusesSaysWhy(t *testing.T) {
	desk := newDesk()
	desk.statements.Err = fmt.Errorf("%w: cannot execute DELETE in a read-only transaction", sqlquery.ErrRefused)

	status, _, answer := desk.ask(t, http.MethodPost, "/sql", "DELETE FROM planet.tiles")

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, answer, "cannot execute DELETE in a read-only transaction")
}

func TestAnEmptyStatementIsNotRun(t *testing.T) {
	desk := newDesk()

	status, _, _ := desk.ask(t, http.MethodPost, "/sql", "  \n")

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Empty(t, desk.statements.Asked)
}

func TestTheJournalOfOneContainerAnswersLineByLine(t *testing.T) {
	desk := newDesk()
	desk.journals.Excerpt = excerpt.Excerpt{Lines: []string{"antibot ban one", "antibot ban two"}, Truncated: true}

	status, header, answer := desk.ask(t, http.MethodGet,
		"/journal/cp-backend?since=2026-10-08T08:00:00Z&until=2026-10-08T09:00:00Z&contains=antibot+ban&limit=2", "")

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "true", header.Get("Ops-Truncated"))
	assert.Equal(t, "antibot ban one\nantibot ban two\n", answer)
	assert.Equal(t, []journal.Query{{
		Tag:      "cp-backend",
		Since:    time.Date(2026, 10, 8, 8, 0, 0, 0, time.UTC),
		Until:    time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC),
		Contains: "antibot ban",
		Limit:    excerpt.Limit{Lines: 2, Bytes: 8 << 20},
	}}, desk.journals.Asked)
}

func TestAJournalWithoutAnEndRunsUntilNow(t *testing.T) {
	desk := newDesk()

	status, header, _ := desk.ask(t, http.MethodGet, "/journal/cp-postgres?since=2026-10-08T08:00:00Z", "")

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "false", header.Get("Ops-Truncated"))
	require.Len(t, desk.journals.Asked, 1)
	assert.WithinDuration(t, time.Now(), desk.journals.Asked[0].Until, time.Minute)
}

func TestTheJournalOfAnythingButThisStackIsRefused(t *testing.T) {
	desk := newDesk()

	status, _, _ := desk.ask(t, http.MethodGet, "/journal/sshd?since=2026-10-08T08:00:00Z", "")

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Empty(t, desk.journals.Asked)
}

func TestAWindowThatCannotBeReadIsRefused(t *testing.T) {
	for _, query := range []string{
		"",
		"since=yesterday",
		"since=2026-10-08T08:00:00Z&until=tomorrow",
		"since=2026-10-08T08:00:00Z&until=2026-10-08T07:00:00Z",
		"since=2026-10-08T08:00:00Z&limit=0",
		"since=2026-10-08T08:00:00Z&limit=20001",
		"since=2026-10-08T08:00:00Z&limit=many",
	} {
		desk := newDesk()

		status, _, _ := desk.ask(t, http.MethodGet, "/caddy?"+query, "")

		assert.Equal(t, http.StatusBadRequest, status, query)
		assert.Empty(t, desk.accessLogs.Asked, query)
	}
}

func TestTheAccessLogAnswersLineByLine(t *testing.T) {
	desk := newDesk()
	desk.accessLogs.Excerpt = excerpt.Excerpt{Lines: []string{`{"ts":1791446460.5,"status":429}`}}

	status, header, answer := desk.ask(t, http.MethodGet,
		"/caddy?since=2026-10-08T08:00:00Z&until=2026-10-08T09:00:00Z&contains=%22status%22%3A429", "")

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "false", header.Get("Ops-Truncated"))
	assert.Equal(t, `{"ts":1791446460.5,"status":429}`+"\n", answer)
	assert.Equal(t, []accesslog.Query{{
		Since:    time.Date(2026, 10, 8, 8, 0, 0, 0, time.UTC),
		Until:    time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC),
		Contains: `"status":429`,
		Limit:    excerpt.Limit{Lines: 1000, Bytes: 8 << 20},
	}}, desk.accessLogs.Asked)
}

func TestAReadThatFailsSaysSo(t *testing.T) {
	desk := newDesk()
	desk.journals.Err = io.ErrUnexpectedEOF

	status, _, answer := desk.ask(t, http.MethodGet, "/journal/cp-backend?since=2026-10-08T08:00:00Z", "")

	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Contains(t, answer, "unexpected EOF")
}
