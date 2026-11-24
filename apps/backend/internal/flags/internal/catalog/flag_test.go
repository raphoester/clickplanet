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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	openedAt = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	closedAt = openedAt.Add(time.Hour)
)

func TestAFlagIDIsRAndTenLettersOrDigitsOrTheCodeOfAFlagThatLeftTheBuild(t *testing.T) {
	for _, valid := range []string{"r-k3p9x2ma0b", "r-0000000001", "xb"} {
		id, err := catalog.ParseFlagID(valid)
		require.NoError(t, err, valid)
		assert.Equal(t, valid, id.String())
	}
	for _, invalid := range []string{"", "fr", "gb-eng", "r-", "r-K3P9X2MA0B", "r-k3p9x2ma0", "r-k3p9x2ma0bc", "Rennes"} {
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

	assert.Equal(t, catalog.Closed, flag.TakeSnapshot().State)
	assert.Equal(t, closedAt, flag.TakeSnapshot().DecidedAt)
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

func TestTimeAddedToAnOpenFlagEndsItLater(t *testing.T) {
	flag := catalog.OpenFlag("r-0000000001", "Rennes", openedAt)
	days, err := catalog.ParseDays(7)
	require.NoError(t, err)

	added, err := flag.TimeAdded(days, openedAt.Add(time.Hour))
	require.NoError(t, err)

	assert.Equal(t, flag.TakeSnapshot().EndsAt.Add(7*24*time.Hour), added.TakeSnapshot().EndsAt)
	assert.Equal(t, catalog.Open, added.TakeSnapshot().State)
	assert.Equal(t, openedAt, added.TakeSnapshot().DecidedAt, "still open since it opened")
}

func TestAFlagClosedComesBackWithinThirtyDaysAndNotAfter(t *testing.T) {
	closed, err := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Closed(closedAt)
	require.NoError(t, err)
	days, err := catalog.ParseDays(30)
	require.NoError(t, err)

	back := closedAt.Add(29 * 24 * time.Hour)
	reopened, err := closed.TimeAdded(days, back)
	require.NoError(t, err)
	kept := reopened.TakeSnapshot()
	assert.Equal(t, catalog.Open, kept.State)
	assert.Equal(t, back, kept.DecidedAt)
	assert.Equal(t, back.Add(30*24*time.Hour), kept.EndsAt)
	assert.Equal(t, closed.TakeSnapshot().ID, kept.ID, "the same code")
	assert.Equal(t, closed.TakeSnapshot().Name, kept.Name, "the same name")
	assert.Equal(t, closed.TakeSnapshot().Picture, kept.Picture, "the same picture")

	_, err = closed.TimeAdded(days, closedAt.Add(30*24*time.Hour))
	require.ErrorIs(t, err, catalog.ErrTooLate)
}

func TestAFlagExpiresAtItsEndAndNotBefore(t *testing.T) {
	flag := catalog.OpenFlag("r-0000000001", "Rennes", openedAt)
	ends := flag.TakeSnapshot().EndsAt

	_, expired := flag.Expired(ends.Add(-time.Second))
	assert.False(t, expired)

	closed, expired := flag.Expired(ends.Add(time.Minute))
	require.True(t, expired)
	assert.Equal(t, catalog.Closed, closed.TakeSnapshot().State)
	assert.Equal(t, ends, closed.TakeSnapshot().DecidedAt, "closed at its end, however late the check")

	_, expired = closed.Expired(ends.Add(time.Hour))
	assert.False(t, expired, "closed once")
}

func TestDaysAreOneToAboutTenYears(t *testing.T) {
	for _, days := range []uint32{1, 30, 3660} {
		_, err := catalog.ParseDays(days)
		require.NoError(t, err, days)
	}
	for _, days := range []uint32{0, 3661} {
		_, err := catalog.ParseDays(days)
		require.ErrorIs(t, err, catalog.ErrInvalidDays, days)
	}
}

func TestAFlagEndingNowOrBeforeIsNotOpened(t *testing.T) {
	opening := catalog.NewOpening(&catalog.Sequence{}, cptime.NewFixedClock(openedAt))
	picture, err := catalog.TestPicture.Restore()
	require.NoError(t, err)

	_, err = opening.Open("Rennes", picture, openedAt)
	require.ErrorIs(t, err, catalog.ErrEnded)

	flag, err := opening.Open("Rennes", picture, openedAt.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, catalog.FlagID("r-0000000001"), flag.TakeSnapshot().ID)
}

func TestARenamedFlagKeepsItsCodeAndTellsItsNewName(t *testing.T) {
	renamed, fact := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Renamed("Roazhon", closedAt)

	assert.Equal(t, catalog.Name("Roazhon"), renamed.TakeSnapshot().Name)
	assert.Equal(t, catalog.FlagID("r-0000000001"), renamed.TakeSnapshot().ID)
	assert.True(t, proto.Equal(&flagsv1.FlagRenamed{Flag: "r-0000000001", Name: "Roazhon", RenamedAt: timestamppb.New(closedAt)},
		fact), "%v", fact)
}
