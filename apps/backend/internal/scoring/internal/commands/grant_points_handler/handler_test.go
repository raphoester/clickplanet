package grant_points_handler_test

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/grant_points_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/commands/grant_points_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/commands/log_command"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/commands/scoring_command_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/log_subscriber"
)

var at = time.Date(2026, 10, 28, 20, 0, 0, 0, time.UTC)

func setUp() (scoring_command_handler.Handler[*scoringv1.GrantPoints], *inmemory_grant_store.Store) {
	store := inmemory_grant_store.New()
	logger := slog.New(slog.DiscardHandler)
	return scoring_command_handler.New(log_subscriber.New[*scoringv1.GrantPoints](
		grant_points_handler.New(log_command.New("grant_points", grant_points_usecase.New(store), logger)), logger)), store
}

func grantPoints(id, country string, points uint32) *scoringv1.GrantPoints {
	return &scoringv1.GrantPoints{GrantId: id, Country: country, Points: points, GrantedAt: timestamppb.New(at)}
}

func TestAGrantToldTwiceIsKeptOnce(t *testing.T) {
	handler, store := setUp()

	require.NoError(t, handler.Handle(t.Context(), grantPoints("grant", "fr", 2)))
	require.NoError(t, handler.Handle(t.Context(), grantPoints("grant", "de", 3)))

	kept, err := store.Since(t.Context(), at)
	require.NoError(t, err)
	assert.Equal(t, []books.Grant{{ID: "grant", Country: "fr", Points: 2, At: at}}, kept)
}

func TestAGrantThatCanNeverApplyIsDropped(t *testing.T) {
	handler, store := setUp()

	for _, command := range []*scoringv1.GrantPoints{
		grantPoints("", "fr", 2),
		grantPoints(strings.Repeat("a", 65), "fr", 2),
		grantPoints("grant", "", 2),
		grantPoints("grant", "fr", 0),
		{GrantId: "grant", Country: "fr", Points: 2},
	} {
		require.NoError(t, handler.Handle(t.Context(), command), "%v", command)
	}

	kept, err := store.Since(t.Context(), time.Time{})
	require.NoError(t, err)
	assert.Empty(t, kept)
}
