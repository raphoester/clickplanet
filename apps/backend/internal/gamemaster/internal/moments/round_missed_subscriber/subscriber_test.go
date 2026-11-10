package round_missed_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/book_missed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/inmemory_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/round_missed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_missed_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_missed_round_usecase"
)

func TestARoundAndABookToldMissedAreKeptOnceAndOneWithNoIdIsDropped(t *testing.T) {
	store := inmemory_moment_store.New()
	rounds := round_missed_subscriber.New(hear_missed_round_usecase.New(store))
	books := book_missed_subscriber.New(hear_missed_book_usecase.New(store))
	ended := time.Date(2026, 11, 10, 21, 0, 0, 0, time.UTC)

	for range 2 {
		require.NoError(t, rounds.Handle(t.Context(), &scoringv1.RoundMissed{RoundId: "season-1/day-10", EndedAt: timestamppb.New(ended)}))
		require.NoError(t, books.Handle(t.Context(), &scoringv1.BookMissed{BookId: "season-2", EndedAt: timestamppb.New(ended)}))
	}
	require.NoError(t, rounds.Handle(t.Context(), &scoringv1.RoundMissed{}))
	require.NoError(t, books.Handle(t.Context(), &scoringv1.BookMissed{}))

	kept, err := store.Kept(t.Context())
	require.NoError(t, err)
	assert.Equal(t, moments.Kept{MissedRounds: []moments.RoundID{"season-1/day-10"}, MissedBooks: []moments.BookID{"season-2"}}, kept)
}
