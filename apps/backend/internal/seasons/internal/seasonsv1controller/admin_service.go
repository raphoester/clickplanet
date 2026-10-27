package seasonsv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/save_rule_set_handler"
)

type AdminService struct {
	get_rules_handler.GetRulesHandler
	save_rule_set_handler.SaveRuleSetHandler
}

var _ seasonsv1connect.AdminServiceHandler = AdminService{}
