package quizzesv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1/quizzesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/get_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/get_rule_set_names_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/play_rule_set_handler"
)

type InternalService struct {
	play_rule_set_handler.PlayRuleSetHandler
	get_rule_set_names_handler.GetRuleSetNamesHandler
	get_rule_set_handler.GetRuleSetHandler
}

var _ quizzesv1connect.InternalServiceHandler = InternalService{}
