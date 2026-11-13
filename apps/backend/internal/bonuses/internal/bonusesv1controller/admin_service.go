package bonusesv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1/bonusesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/save_rule_set_handler"
)

type AdminService struct {
	get_rules_handler.GetRulesHandler
	save_rule_set_handler.SaveRuleSetHandler
}

var _ bonusesv1connect.AdminServiceHandler = AdminService{}
