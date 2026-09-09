package planetv3controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
)

// mapMaxAge lets the edge serve the same chunk to a burst of visitors. The
// websocket carries everything that happens after the chunk was built, so a
// client that starts from a slightly old map converges anyway.
const mapMaxAge = 5

// MapEncoder is the dense map encoding served by GET /map.
type MapEncoder interface {
	EncodeStateBatch(start uint32, end uint32) ([]byte, error)
}

// MapHandler serves the bulk map load, which is deliberately not an RPC: a
// GET can be cached, and the dense body has no field names to speak of.
type MapHandler struct {
	encoder      MapEncoder
	tilesChecker domain.TilesChecker
	logger       logging.Logger
}

var _ http.Handler = (*MapHandler)(nil)

func NewMapHandler(
	encoder MapEncoder,
	tilesChecker domain.TilesChecker,
	logger logging.Logger,
) *MapHandler {
	if logger == nil {
		logger = logging.NewNopLogger()
	}

	return &MapHandler{
		encoder:      encoder,
		tilesChecker: tilesChecker,
		logger:       logger,
	}
}

func (h *MapHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start, err := tileParam(r, "start", 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	end, err := tileParam(r, "end", h.tilesChecker.MaxIndex())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	body, err := h.encoder.EncodeStateBatch(start, end)
	if err != nil {
		http.Error(w, "invalid tile range", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", mapMaxAge))
	if _, err := w.Write(body); err != nil {
		h.logger.Error("failed to write map chunk", lf.Err(err))
	}
}

func tileParam(r *http.Request, name string, fallback uint32) (uint32, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", name)
	}

	return uint32(value), nil
}
