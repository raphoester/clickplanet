package quizzesv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1/quizzesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoSession = errors.New("quizzes require a session; call auth.v1.AuthService/CreateSession first")

// NewSessionInterceptor asks an opening and an answer for the click token a click carries: the opening starts the
// answer's clock, so the caller it starts it for must be the caller that answers.
func NewSessionInterceptor(
	verifier cpconnect.SessionVerifier, clock cptime.Clock, registerer prometheus.Registerer,
) connect.Interceptor {
	checks := promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
		Name: "quiz_session_checks",
		Help: "Quiz openings and answers by the verdict on the session token they carried",
	}, []string{"verdict"})

	return cpconnect.NewSessionInterceptor(
		verifier,
		clock,
		ErrNoSession,
		func(verdict cpconnect.SessionVerdict) { checks.WithLabelValues(string(verdict)).Inc() },
		cpconnect.Attested(quizzesv1connect.QuizServiceOpenQuizProcedure),
		cpconnect.Attested(quizzesv1connect.QuizServiceAnswerQuizProcedure),
	)
}

// NewStreamSessionReader reads the token a stream opens with, and refuses nothing: a signed-in account's stream attends
// the quizzes as that account, any other as its address.
func NewStreamSessionReader(verifier cpconnect.SessionVerifier, clock cptime.Clock) connect.Interceptor {
	return cpconnect.NewSessionReaderInterceptor(verifier, clock,
		cpconnect.Identified(quizzesv1connect.QuizServiceListenForEventsProcedure))
}
