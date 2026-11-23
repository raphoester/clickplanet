package cpflags_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

func TestAFlagOpenedIsPlayableOnceHeard(t *testing.T) {
	catalog := cpflags.NewCatalog(cpflags.NewMemoryStore(), cpflags.Quiet{})

	require.NoError(t, cpflags.NewOpenedHandler(catalog).Handle(t.Context(),
		&flagsv1.FlagOpened{Flag: brittanyCity, Name: "Rennes", OpenedAt: timestamppb.New(opened)}))

	assert.Equal(t, answers{playable: true, known: true}, answersOf(t, catalog, brittanyCity))
}

func TestAFlagClosedIsNoLongerPlayableOnceHeard(t *testing.T) {
	catalog := cpflags.NewOpenCatalog(brittanyCity)

	require.NoError(t, cpflags.NewClosedHandler(catalog).Handle(t.Context(),
		&flagsv1.FlagClosed{Flag: brittanyCity, ClosedAt: timestamppb.New(closed)}))

	assert.Equal(t, answers{known: true}, answersOf(t, catalog, brittanyCity))
}

func TestAFactThatNamesNoFlagIsRefused(t *testing.T) {
	catalog := cpflags.NewCatalog(cpflags.NewMemoryStore(), cpflags.Quiet{})

	require.ErrorIs(t, cpflags.NewOpenedHandler(catalog).Handle(t.Context(), &flagsv1.FlagOpened{}), cpflags.ErrNoFlag)
	require.ErrorIs(t, cpflags.NewClosedHandler(catalog).Handle(t.Context(), &flagsv1.FlagClosed{}), cpflags.ErrNoFlag)
}
