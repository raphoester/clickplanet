package listen_for_events_handler_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/listen_for_events_handler"
)

type recorder struct {
	sent []*bonusesv1.BonusEvent
	err  error
}

func (r *recorder) Send(event *bonusesv1.BonusEvent) error {
	r.sent = append(r.sent, event)
	return r.err
}

func TestASinkFramesABoxOfferedToThisCaller(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Offer: &bonuses.Offer{
			Token:     "a-token",
			Seed:      42,
			Kind:      bonuses.KindSpreadClicks,
			ExpiresAt: time.Unix(0, 0).Add(15 * time.Second),
		},
	}))

	offered := stream.sent[0].GetBonusOffered()
	require.NotNil(t, offered, "expected a bonus_offered case, got %+v", stream.sent[0].GetEvent())
	assert.Equal(t, "a-token", offered.GetToken())
	assert.Equal(t, uint32(42), offered.GetSeed())
	assert.Equal(t, planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS, offered.GetKind())
	assert.Equal(t, int64(15_000), offered.GetExpiresAtUnixMs())
}

func TestASinkFramesAQuizOfferedToThisCaller(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Quiz: &bonuses.QuizOffer{Token: "a-quiz", ExpiresAt: time.Unix(0, 0).Add(20 * time.Second)},
	}))

	quiz := stream.sent[0].GetQuizOffered()
	require.NotNil(t, quiz, "expected a quiz_offered case, got %+v", stream.sent[0].GetEvent())
	assert.Equal(t, "a-quiz", quiz.GetToken())
	assert.Equal(t, int64(20_000), quiz.GetExpiresAtUnixMs())
}

func TestASinkFramesACatchWithNoTokenOnIt(t *testing.T) {
	stream := &recorder{}

	require.NoError(t, listen_for_events_handler.NewSink(stream).Send(listen_for_events_usecase.Event{
		Taken: &bonuses.Taken{CountryID: "jp", Kind: bonuses.KindShields, QuizSubject: "fr"},
	}))

	taken := stream.sent[0].GetBonusTaken()
	require.NotNil(t, taken)
	assert.Equal(t, "jp", taken.GetCountryId())
	assert.Equal(t, planetv1.BonusKind_BONUS_KIND_SHIELDS, taken.GetKind())
	assert.Equal(t, "fr", taken.GetQuizSubjectCountryId())
}

func TestASinkFramesAHeartbeatAndASetTaken(t *testing.T) {
	stream := &recorder{}
	sink := listen_for_events_handler.NewSink(stream)

	require.NoError(t, sink.Send(listen_for_events_usecase.Event{Heartbeat: true}))
	require.NoError(t, sink.Send(listen_for_events_usecase.Event{RulesChanged: true}))

	require.Len(t, stream.sent, 2)
	assert.NotNil(t, stream.sent[0].GetHeartbeat())
	assert.NotNil(t, stream.sent[1].GetRulesChanged())
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
