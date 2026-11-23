package flag_closed_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/flag_closed_subscriber"
)

const rennes = "r-k3p9x2ma0b"

func TestAFlagClosedIsKnownAndNoLongerPlayableOnceHeard(t *testing.T) {
	flags := catalog.NewOpenCatalog(rennes)

	require.NoError(t, flag_closed_subscriber.New(flags).Handle(t.Context(), &flagsv1.FlagClosed{
		Flag: rennes, ClosedAt: timestamppb.New(time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)),
	}))

	playable, err := flags.Playable(t.Context(), rennes)
	require.NoError(t, err)
	assert.False(t, playable)
	known, err := flags.Known(t.Context(), rennes)
	require.NoError(t, err)
	assert.True(t, known, "closed, never forgotten")
}

func TestAFactThatNamesNoFlagIsRefused(t *testing.T) {
	err := flag_closed_subscriber.New(catalog.NewOpenCatalog()).Handle(t.Context(), &flagsv1.FlagClosed{})

	require.ErrorIs(t, err, flag_closed_subscriber.ErrNoFlag)
}
