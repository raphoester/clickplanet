package boxesv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1/boxesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/save_rule_set_handler"
)

type AdminService struct {
	get_rules_handler.GetRulesHandler
	save_rule_set_handler.SaveRuleSetHandler
}

var _ boxesv1connect.AdminServiceHandler = AdminService{}
