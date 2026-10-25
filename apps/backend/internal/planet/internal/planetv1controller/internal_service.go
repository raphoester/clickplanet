package planetv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_allegiance_of_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_grounds_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_takes_by_country_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_territories_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/grant_charge_handler"
)

type InternalService struct {
	get_takes_by_country_handler.GetTakesByCountryHandler
	get_territories_handler.GetTerritoriesHandler
	get_grounds_handler.GetGroundsHandler
	grant_charge_handler.GrantChargeHandler
	get_allegiance_of_handler.GetAllegianceOfHandler
}

var _ planetv1connect.InternalServiceHandler = InternalService{}
