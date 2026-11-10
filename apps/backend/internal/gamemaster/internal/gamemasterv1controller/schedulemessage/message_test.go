package schedulemessage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/schedulemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

func TestEveryKindTheWireNamesIsReadAndWrittenBack(t *testing.T) {
	for wire, kept := range map[gamemasterv1.EntryKind]schedule.Kind{
		gamemasterv1.EntryKind_ENTRY_KIND_BATTLE: schedule.Battle,
		gamemasterv1.EntryKind_ENTRY_KIND_PLAIN:  schedule.Plain,
	} {
		draft, err := schedulemessage.DraftOf("rush", wire, "Triple points.", nil, nil, 1, 3)
		require.NoError(t, err)
		assert.Equal(t, kept, draft.Kind)
		assert.Equal(t, schedule.Description("Triple points."), draft.Description)
		assert.Equal(t, schedule.Times(3), draft.Times)

		written, err := schedulemessage.KindOf(kept)
		require.NoError(t, err)
		assert.Equal(t, wire, written)
	}
}

func TestAKeptKindThisBuildDoesNotKnowIsAnError(t *testing.T) {
	_, err := schedulemessage.KindOf("siege")

	require.ErrorIs(t, err, schedule.ErrUnknownKind)
}
