package scoringv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/save_rule_set_handler"
)

type AdminService struct {
	get_rules_handler.GetRulesHandler
	save_rule_set_handler.SaveRuleSetHandler
}

var _ scoringv1connect.AdminServiceHandler = AdminService{}
