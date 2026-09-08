package clicks_controller

import (
	"net/http"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

func (c *Controller) GetMapDensity(w http.ResponseWriter, _ *http.Request) {
	maxIndex := c.tilesChecker.MaxIndex()
	c.answerer.Data(w, &planetv1.MapDensityResponse{
		Density: maxIndex,
	})
}
