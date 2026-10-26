package antibotv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/ban_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/inspect_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/save_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/unban_player_handler"
)

type AdminService struct {
	ban_player_handler.BanPlayerHandler
	unban_player_handler.UnbanPlayerHandler
	inspect_player_handler.InspectPlayerHandler
	get_rules_handler.GetRulesHandler
	save_rule_set_handler.SaveRuleSetHandler
}

var _ antibotv1connect.AdminServiceHandler = AdminService{}
