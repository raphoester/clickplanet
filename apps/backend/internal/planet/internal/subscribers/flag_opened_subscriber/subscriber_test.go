package flag_opened_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/flag_opened_subscriber"
)

const rennes = "r-k3p9x2ma0b"

func TestAFlagOpenedIsPlayableOnceHeard(t *testing.T) {
	flags := catalog.NewCatalog(catalog.NewMemoryStore(), catalog.Unlocked)

	require.NoError(t, flag_opened_subscriber.New(flags).Handle(t.Context(), &flagsv1.FlagOpened{
		Flag: rennes, Name: "Rennes", OpenedAt: timestamppb.New(time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)),
	}))

	playable, err := flags.Playable(t.Context(), rennes)
	require.NoError(t, err)
	assert.True(t, playable)
}

func TestAFactThatNamesNoFlagIsRefused(t *testing.T) {
	err := flag_opened_subscriber.New(catalog.NewCatalog(catalog.NewMemoryStore(), catalog.Unlocked)).
		Handle(t.Context(), &flagsv1.FlagOpened{})

	require.ErrorIs(t, err, flag_opened_subscriber.ErrNoFlag)
}
