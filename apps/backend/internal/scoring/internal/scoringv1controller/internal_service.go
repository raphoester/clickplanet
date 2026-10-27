package scoringv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_rule_set_names_handler"
)

type InternalService struct {
	get_rule_set_names_handler.GetRuleSetNamesHandler
}

var _ scoringv1connect.InternalServiceHandler = InternalService{}
