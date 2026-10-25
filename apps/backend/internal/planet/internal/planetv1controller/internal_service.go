package planetv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/apply_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/fill_charges_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_bombs_and_fortifies_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_takes_by_country_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_territories_handler"
)

type InternalService struct {
	get_takes_by_country_handler.GetTakesByCountryHandler
	get_territories_handler.GetTerritoriesHandler
	get_bombs_and_fortifies_handler.GetBombsAndFortifiesHandler
	apply_rule_set_handler.ApplyRuleSetHandler
	fill_charges_handler.FillChargesHandler
}

var _ planetv1connect.InternalServiceHandler = InternalService{}
