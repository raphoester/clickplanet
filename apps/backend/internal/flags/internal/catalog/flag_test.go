package catalog_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
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

	_, err = opening.Open("Rennes", picture, catalog.TestPin, openedAt)
	require.ErrorIs(t, err, catalog.ErrEnded)

	flag, err := opening.Open("Rennes", picture, catalog.TestPin, openedAt.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, catalog.FlagID("r-0000000001"), flag.TakeSnapshot().ID)
}

func TestAPinIsAPointOnTheGlobe(t *testing.T) {
	for _, valid := range [][2]float64{{48.1173, -1.6778}, {90, 180}, {-90, -180}, {0, 0}} {
		pin, err := catalog.ParsePin(valid[0], valid[1])
		require.NoError(t, err, valid)
		assert.Equal(t, catalog.Pin{Latitude: valid[0], Longitude: valid[1]}, pin)
	}
	for _, invalid := range [][2]float64{{90.1, 0}, {-91, 0}, {0, 180.1}, {0, -181}, {math.NaN(), 0}, {0, math.NaN()}} {
		_, err := catalog.ParsePin(invalid[0], invalid[1])
		require.ErrorIs(t, err, catalog.ErrInvalidPin, invalid)
	}
}

func TestARestyledFlagWearsItsNewNameAndPictureAndTellsIt(t *testing.T) {
	closed, err := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Closed(closedAt)
	require.NoError(t, err)
	picture, err := pictures.Snapshot{Hash: strings.Repeat("cd", 32), Width: 400, Height: 200}.Restore()
	require.NoError(t, err)

	restyled, fact := closed.Restyled("Breizh", picture, closedAt.Add(time.Hour))

	want := closed.TakeSnapshot()
	want.Name, want.Picture = "Breizh", picture.TakeSnapshot()
	assert.Equal(t, want, restyled.TakeSnapshot(), "closed as it was")
	assert.True(t, proto.Equal(&flagsv1.FlagUpdated{
		Flag: "r-0000000001", Name: "Breizh", UpdatedAt: timestamppb.New(closedAt.Add(time.Hour)),
	}, fact), "%v", fact)
}
