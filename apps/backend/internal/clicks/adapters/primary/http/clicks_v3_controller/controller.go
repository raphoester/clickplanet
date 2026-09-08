package clicks_v3_controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"connectrpc.com/connect"
	clicksv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clicks/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/clicks/v1/clicksv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/click_handler_service"
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

func New(
	clickHandlerService click_handler_service.IService,
	tilesChecker domain.TilesChecker,
	mapEncoder MapEncoder,
	logger logging.Logger,
) *Controller {
	if logger == nil {
		logger = logging.NewNopLogger()
	}

	return &Controller{
		clickHandlerService: clickHandlerService,
		tilesChecker:        tilesChecker,
		mapEncoder:          mapEncoder,
		logger:              logger,
	}
}

type Controller struct {
	clickHandlerService click_handler_service.IService
	tilesChecker        domain.TilesChecker
	mapEncoder          MapEncoder
	logger              logging.Logger
}

var _ clicksv1connect.ClickServiceHandler = (*Controller)(nil)

func (c *Controller) DeclareRoutes(mux *http.ServeMux) {
	path, handler := clicksv1connect.NewClickServiceHandler(c)
	mux.Handle(path, handler)
	mux.HandleFunc("GET /map", c.GetMap)
}

func (c *Controller) Click(
	ctx context.Context,
	req *connect.Request[clicksv1.ClickRequest],
) (*connect.Response[clicksv1.ClickResponse], error) {
	err := c.clickHandlerService.HandleClick(ctx, req.Msg.GetTileId(), req.Msg.GetCountryId())
	if errors.Is(err, domain.ErrInvalidArgument) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		c.logger.Error("failed to handle click", lf.Err(err))
		return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}

	return connect.NewResponse(&clicksv1.ClickResponse{}), nil
}

func (c *Controller) MapDensity(
	_ context.Context,
	_ *connect.Request[clicksv1.MapDensityRequest],
) (*connect.Response[clicksv1.MapDensityResponse], error) {
	return connect.NewResponse(&clicksv1.MapDensityResponse{
		Density: c.tilesChecker.MaxIndex(),
	}), nil
}

// GetMap answers with the dense encoding of a tile range, defaulting to the
// whole map. Unlike the v2 POST it replaced, it is cacheable.
func (c *Controller) GetMap(w http.ResponseWriter, r *http.Request) {
	start, err := tileParam(r, "start", 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	end, err := tileParam(r, "end", c.tilesChecker.MaxIndex())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	body, err := c.mapEncoder.EncodeStateBatch(start, end)
	if err != nil {
		http.Error(w, "invalid tile range", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", mapMaxAge))
	if _, err := w.Write(body); err != nil {
		c.logger.Error("failed to write map chunk", lf.Err(err))
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
