package season_podium_subscriber_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums/usecases/record_podium_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/season_podium_subscriber"
)

type spyUseCase struct {
	told []record_podium_usecase.In
}

func (s *spyUseCase) Execute(_ context.Context, in record_podium_usecase.In) error {
	s.told = append(s.told, in)
	return nil
}

func TestASeasonEndedHasItsPodiumRecordedByItsBook(t *testing.T) {
	spy := &spyUseCase{}

	err := season_podium_subscriber.New(spy).Handle(t.Context(), &gamemasterv1.SeasonEnded{Season: 1, BookId: "season-1"})

	require.NoError(t, err)
	assert.Equal(t, []record_podium_usecase.In{{Season: 1, Book: "season-1"}}, spy.told, "a season with no finale has a podium too")
}

func TestASeasonWithNoBookIsRefused(t *testing.T) {
	spy := &spyUseCase{}

	err := season_podium_subscriber.New(spy).Handle(t.Context(), &gamemasterv1.SeasonEnded{Season: 0})

	require.ErrorIs(t, err, finales.ErrNoBook)
	assert.Empty(t, spy.told)
}
