package planetv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_allegiance_of_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_country_sizes_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_totals_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rule_set_names_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_takes_by_country_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_territories_handler"
)

type InternalService struct {
	get_ledger_handler.GetLedgerHandler
	get_ledger_totals_handler.GetLedgerTotalsHandler
	get_takes_by_country_handler.GetTakesByCountryHandler
	get_territories_handler.GetTerritoriesHandler
	get_country_sizes_handler.GetCountrySizesHandler
	get_allegiance_of_handler.GetAllegianceOfHandler
	get_rule_set_names_handler.GetRuleSetNamesHandler
}

var _ planetv1connect.InternalServiceHandler = InternalService{}
