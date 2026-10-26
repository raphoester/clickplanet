package planetv1controller

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/end_override_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/find_players_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ledger_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_replay_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_territories_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/grant_charges_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/override_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/paint_random_tiles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/reassign_country_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/revert_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/save_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/top_players_handler"
)

type AdminService struct {
	reassign_country_handler.ReassignCountryHandler
	find_players_handler.FindPlayersHandler
	top_players_handler.TopPlayersHandler
	revert_player_handler.RevertPlayerHandler
	paint_random_tiles_handler.PaintRandomTilesHandler
	grant_charges_handler.GrantChargesHandler
	get_replay_handler.GetReplayHandler
	get_ledger_handler.GetLedgerHandler
	get_territories_handler.GetTerritoriesHandler
	get_rules_handler.GetRulesHandler
	save_rule_set_handler.SaveRuleSetHandler
	override_rules_handler.OverrideRulesHandler
	end_override_handler.EndOverrideHandler
}

var _ planetv1connect.AdminServiceHandler = AdminService{}

// The antibot answers these three on their paths until the backoffice calls antibot.v1: a call never reaches here.
var errMoved = errors.New("moved to antibot.v1.AdminService")

func (AdminService) BanPlayer(
	context.Context, *connect.Request[planetv1.BanPlayerRequest],
) (*connect.Response[planetv1.BanPlayerResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errMoved)
}

func (AdminService) UnbanPlayer(
	context.Context, *connect.Request[planetv1.UnbanPlayerRequest],
) (*connect.Response[planetv1.UnbanPlayerResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errMoved)
}

func (AdminService) InspectPlayer(
	context.Context, *connect.Request[planetv1.InspectPlayerRequest],
) (*connect.Response[planetv1.InspectPlayerResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errMoved)
}
