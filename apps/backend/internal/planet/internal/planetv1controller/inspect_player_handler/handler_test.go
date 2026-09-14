package inspect_player_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/inspect_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/inspect_player_handler"
)

type stubUseCase struct {
	in  inspect_player_usecase.In
	out antibot.Examination
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in inspect_player_usecase.In) (antibot.Examination, error) {
	s.in = in
	return s.out, s.err
}

func inspect(t *testing.T, useCase *stubUseCase, scope string) (*planetv1.InspectPlayerResponse, error) {
	t.Helper()

	res, err := inspect_player_handler.New(useCase).InspectPlayer(
		t.Context(), connect.NewRequest(&planetv1.InspectPlayerRequest{Scope: scope}))
	if err != nil {
		return nil, fmt.Errorf("inspect refused: %w", err)
	}

	return res.Msg, nil
}

func TestTheExaminationComesBackWhole(t *testing.T) {
	at := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	useCase := &stubUseCase{out: antibot.Examination{
		Scope:       "9.9.9.9",
		Tracked:     true,
		Banned:      true,
		Flags:       3,
		Offence:     2,
		BannedUntil: at.Add(24 * time.Hour),
		Readings: []antibot.Reading{
			{Watchdog: "metronome", Level: "suspect", Evidence: "cadence clicks=120 spread=40ms", At: at},
			{Watchdog: "sequencer", Level: "clear"},
		},
		Suspects:         1,
		MinSuspects:      2,
		Clicks:           900,
		ActiveFor:        time.Hour,
		LongestGap:       3 * time.Second,
		LastClickAt:      at,
		TopCountry:       "fr",
		TopCountryClicks: 850,
	}}

	res, err := inspect(t, useCase, "9.9.9.9")
	require.NoError(t, err)

	assert.Equal(t, inspect_player_usecase.In{Scope: "9.9.9.9"}, useCase.in)

	assert.Equal(t, "9.9.9.9", res.GetScope())
	assert.True(t, res.GetTracked())
	assert.True(t, res.GetBanned())
	assert.Equal(t, uint32(3), res.GetFlags())
	assert.Equal(t, uint32(2), res.GetOffence())
	assert.Equal(t, at.Add(24*time.Hour), res.GetBannedUntil().AsTime())

	require.Len(t, res.GetReadings(), 2)
	assert.Equal(t, "metronome", res.GetReadings()[0].GetWatchdog())
	assert.Equal(t, "suspect", res.GetReadings()[0].GetLevel())
	assert.Equal(t, "cadence clicks=120 spread=40ms", res.GetReadings()[0].GetEvidence())
	assert.Equal(t, at, res.GetReadings()[0].GetAt().AsTime())
	assert.Equal(t, "clear", res.GetReadings()[1].GetLevel())
	assert.Empty(t, res.GetReadings()[1].GetEvidence())
	assert.Nil(t, res.GetReadings()[1].GetAt())

	assert.Equal(t, uint32(1), res.GetSuspects())
	assert.Equal(t, uint32(2), res.GetMinSuspects())
	require.NotNil(t, res.Guilty)
	assert.False(t, res.GetGuilty())

	assert.Equal(t, uint32(900), res.GetClicks())
	assert.Equal(t, time.Hour, res.GetActiveFor().AsDuration())
	assert.Equal(t, 3*time.Second, res.GetLongestGap().AsDuration())
	assert.Equal(t, at, res.GetLastClickAt().AsTime())
	assert.Equal(t, "fr", res.GetTopCountry())
	assert.Equal(t, uint32(850), res.GetTopCountryClicks())
}

func TestAnUntrackedScopeStillSaysSo(t *testing.T) {
	res, err := inspect(t, &stubUseCase{out: antibot.Examination{Scope: "9.9.9.9", MinSuspects: 2}}, "9.9.9.9")
	require.NoError(t, err)

	require.NotNil(t, res.Tracked)
	assert.False(t, res.GetTracked())
	require.NotNil(t, res.Banned)
	assert.False(t, res.GetBanned())
	assert.Nil(t, res.GetBannedUntil())
	assert.Nil(t, res.GetActiveFor())
	assert.Nil(t, res.GetLastClickAt())
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	cause := errors.New("boom")

	for err, code := range map[error]connect.Code{
		fmt.Errorf("%w: %q", ledger.ErrInvalidScope, "bot"): connect.CodeInvalidArgument,
		inspect_player_usecase.ErrAntiBotOff:                connect.CodeFailedPrecondition,
		cause:                                               connect.CodeUnknown,
	} {
		_, got := inspect(t, &stubUseCase{err: err}, "bot")
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
