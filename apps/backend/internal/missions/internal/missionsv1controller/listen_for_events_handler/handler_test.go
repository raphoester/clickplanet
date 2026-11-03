package listen_for_events_handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1/missionsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/get_missions_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

var (
	ada  = missions.AccountID{15: 1}
	bank = missions.MissionID{15: 1}
)

type stubUseCase struct {
	day   get_missions_usecase.Out
	asked chan missions.AccountID
}

func (s stubUseCase) Execute(_ context.Context, account missions.AccountID, sink listen_for_events_usecase.Sink) error {
	s.asked <- account
	if err := sink.SendDay(s.day); err != nil {
		return err //nolint:wrapcheck // the stream's own error.
	}
	return sink.SendHeartbeat() //nolint:wrapcheck // the stream's own error.
}

func paidDay(t *testing.T) get_missions_usecase.Out {
	t.Helper()

	brief, err := missions.BriefOf([]missions.MissionSnapshot{
		{ID: bank, Order: missions.TakeFor{Flag: "fr", Goal: 60}, Reward: missions.Reward{Kind: missions.RewardShields, Amount: 10}},
	})
	require.NoError(t, err)
	return get_missions_usecase.Out{
		Day: days.TestingDayAt(time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)), Brief: brief,
		Sheet: missions.SheetOf(missions.SheetSnapshot{
			Progress: map[missions.MissionID]uint32{bank: 60}, Paid: cpcolls.NewSet(bank),
		}),
	}
}

func listen(t *testing.T, useCase stubUseCase, signedIn bool) *connect.ServerStreamForClient[missionsv1.MissionEvent] {
	t.Helper()

	handler := listen_for_events_handler.New(useCase)
	mux := http.NewServeMux()
	mux.Handle(missionsv1connect.MissionServiceListenForEventsProcedure, connect.NewServerStreamHandler(
		missionsv1connect.MissionServiceListenForEventsProcedure, handler.ListenForEvents,
	))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if signedIn {
			r = r.WithContext(cpctx.AddAccountToContext(r.Context(), ada.String()))
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	stream, err := missionsv1connect.NewMissionServiceClient(server.Client(), server.URL).ListenForEvents(ctx,
		connect.NewRequest(&missionsv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })
	return stream
}

func TestAStreamWithAnAccountGetsItsDayAsGetMissionsAnswersIt(t *testing.T) {
	useCase := stubUseCase{day: paidDay(t), asked: make(chan missions.AccountID, 1)}
	stream := listen(t, useCase, true)

	require.True(t, stream.Receive(), stream.Err())
	day := stream.Msg().GetDay()
	require.NotNil(t, day, "the day comes first: %v", stream.Msg())
	require.Len(t, day.GetMissions(), 1)
	assert.Equal(t, bank.String(), day.GetMissions()[0].GetId())
	assert.True(t, day.GetMissions()[0].GetPaid())
	assert.Equal(t, time.Date(2026, 10, 26, 21, 0, 0, 0, time.UTC).UnixMilli(), day.GetEndsAtUnixMs())

	require.True(t, stream.Receive(), stream.Err())
	assert.NotNil(t, stream.Msg().GetHeartbeat())
	assert.Equal(t, ada, <-useCase.asked)
}

func TestAStreamWithNoAccountIsRefused(t *testing.T) {
	useCase := stubUseCase{day: paidDay(t), asked: make(chan missions.AccountID, 1)}
	stream := listen(t, useCase, false)

	require.False(t, stream.Receive())
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(stream.Err()))
	assert.Empty(t, useCase.asked)
}
