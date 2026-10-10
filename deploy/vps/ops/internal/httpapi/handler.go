package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-ops/internal/access"
	"github.com/raphoester/clickplanet.lol-ops/internal/accesslog"
	"github.com/raphoester/clickplanet.lol-ops/internal/excerpt"
	"github.com/raphoester/clickplanet.lol-ops/internal/journal"
	"github.com/raphoester/clickplanet.lol-ops/internal/sqlquery"
)

const (
	assertionHeader = "Cf-Access-Jwt-Assertion"
	truncatedHeader = "Ops-Truncated"

	longestStatement = 1 << 20
)

func NewHandler(
	verifier access.Verifier,
	statements sqlquery.Executor,
	journals journal.Reader,
	accessLogs accesslog.Reader,
) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /sql", sqlHandler{statements: statements})
	mux.Handle("GET /journal/{tag}", journalHandler{journals: journals})
	mux.Handle("GET /caddy", accessLogHandler{accessLogs: accessLogs})
	return guard{verifier: verifier, next: mux}
}

type guard struct {
	verifier access.Verifier
	next     http.Handler
}

func (g guard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	caller, err := g.verifier.Verify(r.Context(), r.Header.Get(assertionHeader))
	if err != nil {
		http.Error(w, "no valid Cloudflare Access assertion", http.StatusUnauthorized)
		return
	}
	g.next.ServeHTTP(w, r.WithContext(access.WithCaller(r.Context(), caller)))
}

type sqlHandler struct {
	statements sqlquery.Executor
}

func (h sqlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	limit, err := limitOf(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, longestStatement))
	if err != nil {
		http.Error(w, "the statement is longer than 1 MiB", http.StatusRequestEntityTooLarge)
		return
	}
	statement := strings.TrimSpace(string(body))
	if statement == "" {
		http.Error(w, "the body must be one SQL statement", http.StatusBadRequest)
		return
	}

	result, err := h.statements.Execute(r.Context(), sqlquery.Query{
		Statement: statement,
		MaxRows:   limit.Lines,
		MaxBytes:  limit.Bytes,
	})
	if err != nil {
		http.Error(w, err.Error(), statusOf(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

type journalHandler struct {
	journals journal.Reader
}

func (h journalHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tag, err := journal.ParseTag(r.PathValue("tag"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slice, err := sliceOf(r.URL.Query(), time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	found, err := h.journals.Read(r.Context(), journal.Query{
		Tag:      tag,
		Since:    slice.since,
		Until:    slice.until,
		Contains: slice.contains,
		Limit:    slice.limit,
	})
	if err != nil {
		http.Error(w, err.Error(), statusOf(err))
		return
	}
	writeExcerpt(w, found)
}

type accessLogHandler struct {
	accessLogs accesslog.Reader
}

func (h accessLogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slice, err := sliceOf(r.URL.Query(), time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	found, err := h.accessLogs.Read(r.Context(), accesslog.Query{
		Since:    slice.since,
		Until:    slice.until,
		Contains: slice.contains,
		Limit:    slice.limit,
	})
	if err != nil {
		http.Error(w, err.Error(), statusOf(err))
		return
	}
	writeExcerpt(w, found)
}

func writeExcerpt(w http.ResponseWriter, found excerpt.Excerpt) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set(truncatedHeader, truncation(found.Truncated))
	for _, line := range found.Lines {
		_, _ = io.WriteString(w, line)
		_, _ = io.WriteString(w, "\n")
	}
}

func truncation(truncated bool) string {
	if truncated {
		return "true"
	}
	return "false"
}

func statusOf(err error) int {
	if errors.Is(err, sqlquery.ErrRefused) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
