package bonusesv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1/bonusesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/claim_bonus_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_grant_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/listen_for_events_handler"
)

type BonusService struct {
	claim_bonus_handler.ClaimBonusHandler
	listen_for_events_handler.ListenForEventsHandler
	get_grant_rules_handler.GetGrantRulesHandler
}

var _ bonusesv1connect.BonusServiceHandler = BonusService{}
