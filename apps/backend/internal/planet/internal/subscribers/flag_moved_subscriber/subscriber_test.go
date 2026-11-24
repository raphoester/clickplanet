package flag_moved_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers/flag_moved_subscriber"
)

const rennes = "r-k3p9x2ma0b"

var opened = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)

func TestAFlagMovedSitsOnItsNewPinAtTheNextArena(t *testing.T) {
	flags := catalog.NewCatalog(catalog.NewMemoryStore(), catalog.Unlocked)
	require.NoError(t, flags.Open(t.Context(), rennes, opened))
	require.NoError(t, flags.Move(t.Context(), rennes, clicks.Pin{Latitude: 48.1173, Longitude: -1.6778}, opened))

	require.NoError(t, flag_moved_subscriber.New(flags).Handle(t.Context(), &flagsv1.FlagMoved{
		Flag: rennes, Pin: &flagsv1.Pin{Latitude: 47.996, Longitude: -4.102}, MovedAt: timestamppb.New(opened.Add(time.Hour)),
	}))

	pinned, err := flags.Pinned(t.Context())
	require.NoError(t, err)
	assert.Contains(t, pinned, clicks.Pinned{Flag: rennes, Pin: clicks.Pin{Latitude: 47.996, Longitude: -4.102}})
}

func TestAFactThatNamesNoPinIsRefused(t *testing.T) {
	err := flag_moved_subscriber.New(catalog.NewCatalog(catalog.NewMemoryStore(), catalog.Unlocked)).
		Handle(t.Context(), &flagsv1.FlagMoved{Flag: rennes})

	require.ErrorIs(t, err, flag_moved_subscriber.ErrNoPin)
}
