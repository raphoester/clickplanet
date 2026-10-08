package planetv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_shares_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_takes_by_country_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/set_rules_handler"
)

type InternalService struct {
	get_takes_by_country_handler.GetTakesByCountryHandler
	set_rules_handler.SetRulesHandler
	get_shares_handler.GetSharesHandler
}

var _ planetv1connect.InternalServiceHandler = InternalService{}
