package season_ended_subscriber_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales/usecases/record_finale_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/season_ended_subscriber"
)

type spyUseCase struct {
	told []record_finale_usecase.In
}

func (s *spyUseCase) Execute(_ context.Context, in record_finale_usecase.In) error {
	s.told = append(s.told, in)
	return nil
}

func TestASeasonEndedWithAFinaleIsRecordedByItsIDs(t *testing.T) {
	spy := &spyUseCase{}

	err := season_ended_subscriber.New(spy).Handle(t.Context(), &gamemasterv1.SeasonEnded{
		Season: 0, BookId: "season-0", FinaleRoundId: "season-0/finale",
	})

	require.NoError(t, err)
	assert.Equal(t, []record_finale_usecase.In{{Season: 0, Book: "season-0", Finale: "season-0/finale"}}, spy.told)
}

func TestASeasonWithNoFinaleHadNobodyFightIt(t *testing.T) {
	spy := &spyUseCase{}

	require.NoError(t, season_ended_subscriber.New(spy).Handle(t.Context(), &gamemasterv1.SeasonEnded{Season: 1, BookId: "season-1"}))

	assert.Empty(t, spy.told)
}

func TestASeasonWithNoBookIsRefused(t *testing.T) {
	spy := &spyUseCase{}

	err := season_ended_subscriber.New(spy).Handle(t.Context(), &gamemasterv1.SeasonEnded{FinaleRoundId: "season-0/finale"})

	require.ErrorIs(t, err, finales.ErrNoBook)
	assert.Empty(t, spy.told)
}
