// Package admin_server serves operator tools on a loopback listener, never on the router Caddy forwards to.
package admin_server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country"
)

const ReassignPath = "/admin/reassign-country"

const maxRequestBytes = 1 << 16

type ReassignUseCase interface {
	Execute(ctx context.Context, in reassign_country.In) (reassign_country.Out, error)
}

type reassignRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	DryRun bool   `json:"dryRun"`
}

type reassignResponse struct {
	From       string `json:"from"`
	To         string `json:"to"`
	DryRun     bool   `json:"dryRun"`
	FromBefore int    `json:"fromBefore"`
	ToBefore   int    `json:"toBefore"`
	Moved      int    `json:"moved"`
	FromAfter  int    `json:"fromAfter"`
	ToAfter    int    `json:"toAfter"`
	Error      string `json:"error,omitempty"`
}

func NewHandler(reassign ReassignUseCase, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST "+ReassignPath, func(w http.ResponseWriter, r *http.Request) {
		var req reassignRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, reassignResponse{Error: "body must be {\"from\", \"to\", \"dryRun\"}: " + err.Error()})
			return
		}

		out, err := reassign.Execute(r.Context(), reassign_country.In{From: req.From, To: req.To, DryRun: req.DryRun})

		res := reassignResponse{
			From: req.From, To: req.To, DryRun: req.DryRun,
			FromBefore: out.FromBefore, ToBefore: out.ToBefore,
			Moved: out.Moved, FromAfter: out.FromAfter, ToAfter: out.ToAfter,
		}

		// Logged either way: it is the only record these tiles did not change hands through play.
		attrs := []any{
			slog.String("from", req.From), slog.String("to", req.To), slog.Bool("dryRun", req.DryRun),
			slog.Int("fromBefore", out.FromBefore), slog.Int("toBefore", out.ToBefore),
			slog.Int("moved", out.Moved), slog.Int("fromAfter", out.FromAfter), slog.Int("toAfter", out.ToAfter),
		}

		if err != nil {
			res.Error = err.Error()
			logger.Warn("admin country reassignment failed", append(attrs, slog.String("error", err.Error()))...)
			writeJSON(w, statusOf(err), res)
			return
		}

		logger.Warn("admin country reassignment", attrs...)
		writeJSON(w, http.StatusOK, res)
	})

	return mux
}

func statusOf(err error) int {
	if errors.Is(err, clicks.ErrUnknownCountry) || errors.Is(err, reassign_country.ErrSameCountry) {
		return http.StatusBadRequest
	}

	return http.StatusInternalServerError
}

func writeJSON(w http.ResponseWriter, status int, body reassignResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
