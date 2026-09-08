package clicks_controller

import (
	"fmt"
	"net/http"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

func (c *Controller) GetOwnershipsByBatch(w http.ResponseWriter, r *http.Request) {
	req := &planetv1.OwnershipBatchRequest{}
	if err := c.reader.Read(r, req); err != nil {
		c.answerer.Err(w,
			fmt.Errorf("failed reading req: %w", err),
			"invalid request",
			http.StatusBadRequest,
		)
		return
	}

	endTileId := req.GetEndTileId()
	if endTileId > c.tilesChecker.MaxIndex() {
		endTileId = c.tilesChecker.MaxIndex()
	}

	if !c.tilesChecker.CheckTile(req.GetStartTileId()) ||
		req.GetStartTileId() > endTileId {

		c.answerer.Err(w,
			fmt.Errorf("invalid tile range %d %d", req.GetStartTileId(), endTileId),
			"invalid tile", http.StatusBadRequest,
		)
		return
	}

	tiles, err := c.tilesStorage.GetStateBatch(r.Context(), req.GetStartTileId(), req.GetEndTileId())
	if err != nil {
		c.answerer.Err(w, fmt.Errorf("failed to get tile values: %w", err),
			"internal error", http.StatusInternalServerError)
		return
	}

	response := &planetv1.Ownerships{
		Bindings: tiles,
	}

	c.answerer.Data(w, response)
}
