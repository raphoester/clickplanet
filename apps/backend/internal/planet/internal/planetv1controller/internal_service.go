package planetv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/close_arena_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_allegiance_of_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_country_sizes_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_freeze_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_totals_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_map_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_room_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rule_set_names_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_takes_by_country_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_territories_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_tiles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/internal_grant_charges_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/open_arena_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/play_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/reset_map_handler"
)

type InternalService struct {
	play_rule_set_handler.PlayRuleSetHandler
	reset_map_handler.ResetMapHandler
	open_arena_handler.OpenArenaHandler
	close_arena_handler.CloseArenaHandler
	get_ledger_handler.GetLedgerHandler
	get_ledger_totals_handler.GetLedgerTotalsHandler
	get_map_history_handler.GetMapHistoryHandler
	get_takes_by_country_handler.GetTakesByCountryHandler
	get_territories_handler.GetTerritoriesHandler
	get_country_sizes_handler.GetCountrySizesHandler
	get_allegiance_of_handler.GetAllegianceOfHandler
	get_rule_set_names_handler.GetRuleSetNamesHandler
	get_rule_set_handler.GetRuleSetHandler
	internal_grant_charges_handler.GrantChargesHandler
	get_room_handler.GetRoomHandler
	get_freeze_handler.GetFreezeHandler
	get_tiles_handler.GetTilesHandler
}

var _ planetv1connect.InternalServiceHandler = InternalService{}
