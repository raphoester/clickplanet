package catalog_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
)

func TestAnAnthemIsARecordingOfTheGamesLibraryOrNone(t *testing.T) {
	germany, err := catalog.ParseAnthem("de")
	require.NoError(t, err)
	assert.Equal(t, "de", germany.Library())

	none, err := catalog.ParseAnthem("")
	require.NoError(t, err)
	assert.True(t, none.IsZero())

	for _, invalid := range []string{"DE", "zz", "xb", "r-jz9cwqzp0k", "Das Lied der Deutschen"} {
		_, err := catalog.ParseAnthem(invalid)
		require.ErrorIs(t, err, catalog.ErrInvalidAnthem, invalid)
	}
}

func TestItsHolderAloneSetsAFlagsAnthemAndEachChangeIsTold(t *testing.T) {
	flag := catalog.HeldFlag("r-jz9cwqzp0k", "Heiliges Römisches Reich", catalog.TestHolder, openedAt)
	germany, err := catalog.ParseAnthem("de")
	require.NoError(t, err)

	sung, fact, err := flag.Sung(catalog.TestHolder, germany, closedAt)
	require.NoError(t, err)
	assert.Equal(t, "de", sung.TakeSnapshot().Anthem)
	assert.True(t, proto.Equal(&flagsv1.AnthemChanged{Flag: "r-jz9cwqzp0k", ChangedAt: timestamppb.New(closedAt)}, fact))

	again, fact, err := sung.Sung(catalog.TestHolder, germany, closedAt.Add(time.Minute))
	require.NoError(t, err)
	assert.Nil(t, fact, "the same anthem again tells nothing")
	assert.Equal(t, sung, again)

	off, fact, err := sung.Sung(catalog.TestHolder, catalog.Anthem{}, closedAt.Add(time.Hour))
	require.NoError(t, err)
	assert.NotNil(t, fact)
	assert.Empty(t, off.TakeSnapshot().Anthem, "taken off")

	_, _, err = flag.Sung(catalog.AccountID{15: 9}, germany, closedAt)
	require.ErrorIs(t, err, catalog.ErrNotHolder, "another account")
	_, _, err = catalog.OpenFlag("xb", "Brittany", openedAt).Sung(catalog.AccountID{}, germany, closedAt)
	require.ErrorIs(t, err, catalog.ErrNotHolder, "a flag opened by hand has no holder to choose")
}

func TestAClosedFlagsHolderSetsItsAnthemAndAClaimKeepsIt(t *testing.T) {
	austria, err := catalog.ParseAnthem("at")
	require.NoError(t, err)
	flag, ended := catalog.HeldFlag("r-jz9cwqzp0k", "Heiliges Römisches Reich", catalog.TestHolder, openedAt).
		Expired(openedAt.Add(31 * 24 * time.Hour))
	require.True(t, ended)

	sung, _, err := flag.Sung(catalog.TestHolder, austria, openedAt.Add(32*24*time.Hour))
	require.NoError(t, err)
	claimed, _, err := sung.Paid(catalog.AccountID{15: 9}, openedAt.Add(33*24*time.Hour), catalog.Days(30),
		openedAt.Add(33*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, "at", claimed.TakeSnapshot().Anthem)

	removed, err := sung.Removed("Broke the rules.", openedAt.Add(34*24*time.Hour))
	require.NoError(t, err)
	_, _, err = removed.Sung(catalog.TestHolder, catalog.Anthem{}, openedAt.Add(35*24*time.Hour))
	require.ErrorIs(t, err, catalog.ErrRemoved)
}

func TestAFlagKeptWithAnAnthemThatIsNoRecordingIsRefused(t *testing.T) {
	kept := catalog.HeldFlag("r-jz9cwqzp0k", "Heiliges Römisches Reich", catalog.TestHolder, openedAt).TakeSnapshot()
	kept.Anthem = "zz"
	_, err := kept.Restore()
	require.ErrorIs(t, err, catalog.ErrInvalidAnthem)
}
