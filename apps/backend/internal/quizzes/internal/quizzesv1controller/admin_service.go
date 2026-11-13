package quizzesv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1/quizzesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/get_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/list_quiz_questions_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/save_rule_set_handler"
)

type AdminService struct {
	get_rules_handler.GetRulesHandler
	save_rule_set_handler.SaveRuleSetHandler
	list_quiz_questions_handler.ListQuizQuestionsHandler
}

var _ quizzesv1connect.AdminServiceHandler = AdminService{}
