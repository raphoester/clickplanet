package bonusesv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1/bonusesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rule_set_names_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/play_rule_set_handler"
)

type InternalService struct {
	play_rule_set_handler.PlayRuleSetHandler
	get_rule_set_names_handler.GetRuleSetNamesHandler
	get_rule_set_handler.GetRuleSetHandler
}

var _ bonusesv1connect.InternalServiceHandler = InternalService{}
