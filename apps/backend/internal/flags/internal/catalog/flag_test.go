package catalog_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
)

var (
	openedAt = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	closedAt = openedAt.Add(time.Hour)
)

func TestAFlagIDIsRAndTenLettersOrDigits(t *testing.T) {
	for _, valid := range []string{"r-k3p9x2ma0b", "r-0000000001"} {
		id, err := catalog.ParseFlagID(valid)
		require.NoError(t, err, valid)
		assert.Equal(t, valid, id.String())
	}
	for _, invalid := range []string{"", "fr", "gb-eng", "xb", "r-", "r-K3P9X2MA0B", "r-k3p9x2ma0", "r-k3p9x2ma0bc", "Rennes"} {
		_, err := catalog.ParseFlagID(invalid)
		require.ErrorIs(t, err, catalog.ErrInvalidFlagID, invalid)
	}
}

func TestANameIsTrimmedAndShort(t *testing.T) {
	name, err := catalog.ParseName("  Rennes  ")
	require.NoError(t, err)
	assert.Equal(t, catalog.Name("Rennes"), name)

	_, err = catalog.ParseName(strings.Repeat("é", 40))
	require.NoError(t, err, "40 characters, however many bytes")

	for _, invalid := range []string{"", "   ", strings.Repeat("a", 41), "Ren\nnes", "\xff"} {
		_, err := catalog.ParseName(invalid)
		require.ErrorIs(t, err, catalog.ErrInvalidName, "%q", invalid)
	}
}

func TestAFlagOpenedTellsItIsOpen(t *testing.T) {
	flag := catalog.OpenFlag("r-0000000001", "Rennes", openedAt)

	assertFacts(t, []proto.Message{
		&flagsv1.FlagOpened{Flag: "r-0000000001", Name: "Rennes", OpenedAt: timestamppb.New(openedAt)},
	}, flag.Facts())
}

func TestAFlagClosedTellsItIsClosed(t *testing.T) {
	flag, err := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Closed(closedAt)
	require.NoError(t, err)

	assert.Equal(t, catalog.Snapshot{ID: "r-0000000001", Name: "Rennes", State: catalog.Closed, DecidedAt: closedAt},
		flag.TakeSnapshot())
	assertFacts(t, []proto.Message{
		&flagsv1.FlagClosed{Flag: "r-0000000001", ClosedAt: timestamppb.New(closedAt)},
	}, flag.Facts())
}

func TestAFlagIsClosedOnce(t *testing.T) {
	flag, err := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Closed(closedAt)
	require.NoError(t, err)

	_, err = flag.Closed(closedAt.Add(time.Hour))
	require.ErrorIs(t, err, catalog.ErrClosed)
}

func TestASnapshotOfNoStateDoesNotRestore(t *testing.T) {
	_, err := catalog.Snapshot{ID: "r-0000000001", Name: "Rennes", State: "sold", DecidedAt: openedAt}.Restore()
	require.Error(t, err)
}

func assertFacts(t *testing.T, want []proto.Message, got []proto.Message) {
	t.Helper()
	require.Len(t, got, len(want))
	for i := range want {
		assert.True(t, proto.Equal(want[i], got[i]), "%v", got[i])
	}
}
