package listen_for_events_handler_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzesv1controller/listen_for_events_handler"
)

type recorder struct {
	sent []*quizzesv1.QuizEvent
	err  error
}

func (r *recorder) Send(event *quizzesv1.QuizEvent) error {
	r.sent = append(r.sent, event)
	return r.err
}

func TestASinkFramesAQuizOfferedToThisCaller(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Offer: &quizzes.Offer{Token: "a-quiz", ExpiresAt: time.Unix(0, 0).Add(20 * time.Second)},
	}))

	quiz := stream.sent[0].GetQuizOffered()
	require.NotNil(t, quiz, "expected a quiz_offered case, got %+v", stream.sent[0].GetEvent())
	assert.Equal(t, "a-quiz", quiz.GetToken())
	assert.Equal(t, int64(20_000), quiz.GetExpiresAtUnixMs())
}

func TestASinkFramesAQuizWonWithItsSubject(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Won: &quizzes.Won{CountryID: "jp", Subject: "fr"},
	}))

	won := stream.sent[0].GetQuizWon()
	require.NotNil(t, won)
	assert.Equal(t, "jp", won.GetCountryId())
	assert.Equal(t, "fr", won.GetSubjectCountryId())
}

func TestASinkFramesAHeartbeat(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{Heartbeat: true}))

	require.Len(t, stream.sent, 1)
	assert.NotNil(t, stream.sent[0].GetHeartbeat())
}

func TestASinkSendsNothingForAnEmptyEvent(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{}))
	assert.Empty(t, stream.sent)
}

func TestASinkReportsAFailedSend(t *testing.T) {
	stream := &recorder{err: errors.New("gone")}

	require.Error(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{Heartbeat: true}))
}
