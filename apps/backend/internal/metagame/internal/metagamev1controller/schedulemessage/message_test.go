package schedulemessage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/schedulemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule"
)

func TestEveryKindTheWireNamesIsReadAndWrittenBack(t *testing.T) {
	for wire, kept := range map[metagamev1.EntryKind]schedule.Kind{
		metagamev1.EntryKind_ENTRY_KIND_BATTLE: schedule.Battle,
		metagamev1.EntryKind_ENTRY_KIND_PLAIN:  schedule.Plain,
	} {
		draft, err := schedulemessage.DraftOf("rush", wire, "Triple points.", nil, nil, 1)
		require.NoError(t, err)
		assert.Equal(t, kept, draft.Kind)
		assert.Equal(t, schedule.Description("Triple points."), draft.Description)

		written, err := schedulemessage.KindOf(kept)
		require.NoError(t, err)
		assert.Equal(t, wire, written)
	}
}

func TestAKeptKindThisBuildDoesNotKnowIsAnError(t *testing.T) {
	_, err := schedulemessage.KindOf("siege")

	require.ErrorIs(t, err, schedule.ErrUnknownKind)
}
