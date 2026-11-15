package bonusesv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1/bonusesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_grant_rules_handler"
)

type BonusService struct {
	get_grant_rules_handler.GetGrantRulesHandler
}

var _ bonusesv1connect.BonusServiceHandler = BonusService{}
