package quizzesv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1/quizzesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/answer_quiz_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/open_quiz_handler"
)

type QuizService struct {
	open_quiz_handler.OpenQuizHandler
	answer_quiz_handler.AnswerQuizHandler
	listen_for_events_handler.ListenForEventsHandler
}

var _ quizzesv1connect.QuizServiceHandler = QuizService{}
