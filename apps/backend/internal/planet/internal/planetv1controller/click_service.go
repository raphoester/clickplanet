package planetv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/answer_quiz_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/claim_bonus_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/click_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/drop_bomb_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_bonus_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_budget_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_charges_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_map_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/map_density_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/open_quiz_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/place_shield_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/use_refill_handler"
)

type ClickService struct {
	click_handler.ClickHandler
	get_budget_handler.GetBudgetHandler
	map_density_handler.MapDensityHandler
	get_map_handler.GetMapHandler
	listen_for_events_handler.ListenForEventsHandler
	claim_bonus_handler.ClaimBonusHandler
	drop_bomb_handler.DropBombHandler
	get_charges_handler.GetChargesHandler
	get_bonus_rules_handler.GetBonusRulesHandler
	use_refill_handler.UseRefillHandler
	open_quiz_handler.OpenQuizHandler
	answer_quiz_handler.AnswerQuizHandler
	place_shield_handler.PlaceShieldHandler
}

var _ planetv1connect.ClickServiceHandler = ClickService{}
