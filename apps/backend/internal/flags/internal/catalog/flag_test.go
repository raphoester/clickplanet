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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	openedAt = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	closedAt = openedAt.Add(time.Hour)
)

func TestAFlagIDIsRAndTenLettersOrDigitsOrTheCodeOfAFlagThatLeftTheBuild(t *testing.T) {
	for _, valid := range []string{"r-k3p9x2ma0b", "r-0000000001", "xb", "gb-eng"} {
		id, err := catalog.ParseFlagID(valid)
		require.NoError(t, err, valid)
		assert.Equal(t, valid, id.String())
	}
	for _, invalid := range []string{"", "fr", "gb", "r-", "r-K3P9X2MA0B", "r-k3p9x2ma0", "r-k3p9x2ma0bc", "Rennes"} {
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
		&flagsv1.FlagOpened{Flag: "r-0000000001", Name: "Rennes", OpenedAt: timestamppb.New(openedAt), Pin: &flagsv1.Pin{Latitude: 48.1173, Longitude: -1.6778}},
	}, flag.Facts())
}

func TestAFlagRemovedTellsItIsClosed(t *testing.T) {
	flag, err := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Removed("Broke the rules.", closedAt)
	require.NoError(t, err)

	assert.Equal(t, catalog.Removed, flag.TakeSnapshot().State)
	assert.Equal(t, closedAt, flag.TakeSnapshot().DecidedAt)
	assert.Equal(t, catalog.Note("Broke the rules."), flag.TakeSnapshot().Note, "its holder reads why")
	assertFacts(t, []proto.Message{
		&flagsv1.FlagClosed{Flag: "r-0000000001", ClosedAt: timestamppb.New(closedAt)},
	}, flag.Facts())
}

func TestAFlagClosedAtItsEndIsRemovedAndToldClosedAgain(t *testing.T) {
	ended, ok := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Expired(openedAt.AddDate(0, 2, 0))
	require.True(t, ok)

	removed, err := ended.Removed("Broke the rules.", openedAt.AddDate(0, 3, 0))
	require.NoError(t, err)

	assert.Equal(t, catalog.Removed, removed.TakeSnapshot().State)
	assertFacts(t, []proto.Message{
		&flagsv1.FlagClosed{Flag: "r-0000000001", ClosedAt: timestamppb.New(openedAt.AddDate(0, 3, 0))},
	}, removed.Facts())
}

func TestAFlagIsRemovedOnce(t *testing.T) {
	flag, err := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Removed("Broke the rules.", closedAt)
	require.NoError(t, err)

	_, err = flag.Removed("Broke the rules.", closedAt.Add(time.Hour))
	require.ErrorIs(t, err, catalog.ErrRemoved)
}

func TestAFlagIsNotRemovedWithNoNote(t *testing.T) {
	_, err := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Removed("", closedAt)
	require.ErrorIs(t, err, catalog.ErrInvalidNote)
}

func TestANoteIsOneTo500CharactersWithTheSpacesAtItsEndsCut(t *testing.T) {
	note, err := catalog.ParseNote(" A club's crest.\nNot a place. ")
	require.NoError(t, err)
	assert.Equal(t, catalog.Note("A club's crest.\nNot a place."), note)
	_, err = catalog.ParseNote(strings.Repeat("é", 500))
	require.NoError(t, err)
	_, err = catalog.ParseNote(strings.Repeat("é", 501))
	require.ErrorIs(t, err, catalog.ErrInvalidNote)
	_, err = catalog.ParseNote(" \n ")
	require.ErrorIs(t, err, catalog.ErrInvalidNote)
}

func TestASnapshotOfNoStateDoesNotRestore(t *testing.T) {
	_, err := catalog.Snapshot{ID: "r-0000000001", Name: "Rennes", State: "sold", DecidedAt: openedAt}.Restore()
	require.Error(t, err)
}

func TestASnapshotOfAnOpenFlagWithANoteDoesNotRestore(t *testing.T) {
	kept := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).TakeSnapshot()
	kept.Note = "Broke the rules."
	_, err := kept.Restore()
	require.Error(t, err)
}

func TestASnapshotOfARemovedFlagWithNoNoteDoesNotRestore(t *testing.T) {
	kept := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).TakeSnapshot()
	kept.State = catalog.Removed
	_, err := kept.Restore()
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

	added, err := flag.TimeAdded(days, catalog.NamesOf(nil), openedAt.Add(time.Hour))
	require.NoError(t, err)

	assert.Equal(t, flag.TakeSnapshot().EndsAt.Add(7*24*time.Hour), added.TakeSnapshot().EndsAt)
	assert.Equal(t, catalog.Open, added.TakeSnapshot().State)
	assert.Equal(t, openedAt, added.TakeSnapshot().DecidedAt, "still open since it opened")
}

func TestAFlagClosedAtItsEndComesBackHoweverLongAgo(t *testing.T) {
	closed, ok := catalog.HeldFlag("r-0000000001", "Rennes", catalog.TestHolder, openedAt).Expired(openedAt.AddDate(0, 2, 0))
	require.True(t, ok)

	back := openedAt.AddDate(2, 0, 0)
	reopened, err := closed.TimeAdded(catalog.Days(30), catalog.NamesOf(nil), back)
	require.NoError(t, err)
	kept := reopened.TakeSnapshot()
	assert.Equal(t, catalog.Open, kept.State)
	assert.Equal(t, back, kept.DecidedAt)
	assert.Equal(t, back.Add(30*24*time.Hour), kept.EndsAt)
	assert.Equal(t, closed.TakeSnapshot().ID, kept.ID, "the same code")
	assert.Equal(t, closed.TakeSnapshot().Name, kept.Name, "the same name")
	assert.Equal(t, closed.TakeSnapshot().Picture, kept.Picture, "the same picture")
	assert.Equal(t, catalog.TestHolder, kept.Holder, "the same holder")
}

func TestARemovedFlagComesBackByHandUnlessANewFlagTookItsName(t *testing.T) {
	removed, err := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Removed("Broke the rules.", closedAt)
	require.NoError(t, err)

	back, err := removed.TimeAdded(catalog.Days(7), catalog.NamesOf([]catalog.Flag{removed}), closedAt.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, catalog.Open, back.TakeSnapshot().State)
	assert.Empty(t, back.TakeSnapshot().Note, "the removal it came back from is over")

	taken := catalog.NamesOf([]catalog.Flag{removed, catalog.OpenFlag("r-0000000002", "RENNES", closedAt)})
	_, err = removed.TimeAdded(catalog.Days(7), taken, closedAt.Add(time.Hour))
	require.ErrorIs(t, err, catalog.ErrNameTaken)
}

var (
	ada = catalog.TestHolder
	bob = catalog.AccountID{15: 2}
	// Rennes, held by Ada, ends 30 days after it opened.
	endedAt = openedAt.Add(30 * 24 * time.Hour)
)

func ended(t *testing.T) catalog.Flag {
	t.Helper()
	flag, ok := catalog.HeldFlag("r-0000000001", "Rennes", ada, openedAt).Expired(endedAt.Add(time.Minute))
	require.True(t, ok)
	return flag
}

func TestItsHolderPaysForAnOpenFlagAndNobodyElse(t *testing.T) {
	flag := catalog.HeldFlag("r-0000000001", "Rennes", ada, openedAt)

	require.NoError(t, flag.PayableBy(ada))
	require.ErrorIs(t, flag.PayableBy(bob), catalog.ErrHeld)
	require.ErrorIs(t, catalog.OpenFlag("r-0000000002", "Corsica", openedAt).PayableBy(bob), catalog.ErrHeld,
		"a flag opened by hand is the operator's")

	paid, added, err := flag.Paid(ada, openedAt, catalog.Days(7), openedAt.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 7*24*time.Hour, added)
	assert.Equal(t, endedAt.Add(7*24*time.Hour), paid.TakeSnapshot().EndsAt)

	_, _, err = flag.Paid(bob, openedAt, catalog.Days(7), openedAt.Add(time.Hour))
	require.ErrorIs(t, err, catalog.ErrHeld)
}

func TestAnyAccountClaimsAFlagWhoseTimeEndedHoweverLongAgo(t *testing.T) {
	flag := ended(t)
	require.NoError(t, flag.PayableBy(bob))

	now := endedAt.AddDate(3, 0, 0)
	claimed, added, err := flag.Paid(bob, now.Add(-time.Minute), catalog.Days(7), now)
	require.NoError(t, err)

	kept := claimed.TakeSnapshot()
	assert.Equal(t, catalog.Open, kept.State)
	assert.Equal(t, bob, kept.Holder, "the payer holds it")
	assert.Equal(t, now, kept.DecidedAt)
	assert.Equal(t, now.Add(7*24*time.Hour), kept.EndsAt)
	assert.Equal(t, 7*24*time.Hour, added)
	assert.Equal(t, flag.TakeSnapshot().Name, kept.Name, "the same flag")
	assertFacts(t, []proto.Message{&flagsv1.FlagOpened{
		Flag: "r-0000000001", Name: "Rennes", OpenedAt: timestamppb.New(now), Pin: &flagsv1.Pin{Latitude: 48.1173, Longitude: -1.6778},
	}}, claimed.Facts())
}

func TestAnOrderPlacedBeforeTheFlagEndedClaimsNothing(t *testing.T) {
	flag := ended(t)

	_, _, err := flag.Paid(bob, endedAt.Add(-time.Hour), catalog.Days(7), endedAt.Add(time.Hour))
	require.ErrorIs(t, err, catalog.ErrOrderedBefore, "a late renewal of a plan bought before")

	back, _, err := flag.Paid(ada, openedAt, catalog.Days(7), endedAt.Add(time.Hour))
	require.NoError(t, err, "its holder's late renewal brings it back")
	assert.Equal(t, ada, back.TakeSnapshot().Holder)
}

func TestNobodyPaysForARemovedFlag(t *testing.T) {
	removed, err := ended(t).Removed("Broke the rules.", endedAt.Add(time.Hour))
	require.NoError(t, err)

	require.ErrorIs(t, removed.PayableBy(ada), catalog.ErrRemoved)
	require.ErrorIs(t, removed.PayableBy(bob), catalog.ErrRemoved)
	_, _, err = removed.Paid(ada, endedAt.Add(2*time.Hour), catalog.Days(7), endedAt.Add(2*time.Hour))
	require.ErrorIs(t, err, catalog.ErrRemoved)
}

func TestARefundTakesBackOnlyTimeGrantedSinceTheFlagLastOpened(t *testing.T) {
	now := endedAt.AddDate(0, 1, 0)
	claimed, _, err := ended(t).Paid(bob, now, catalog.Days(30), now)
	require.NoError(t, err)

	kept := claimed.Shortened(30*24*time.Hour, openedAt, now.Add(time.Hour))
	assert.Equal(t, claimed.TakeSnapshot(), kept.TakeSnapshot(), "Ada's month was spent before Bob's claim")

	cut := claimed.Shortened(30*24*time.Hour, now, now.Add(time.Hour))
	assert.Equal(t, catalog.Closed, cut.TakeSnapshot().State, "Bob's own payment refunded")
}

func TestARemovedFlagsNameIsFree(t *testing.T) {
	removed, err := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Removed("Broke the rules.", closedAt)
	require.NoError(t, err)

	require.NoError(t, catalog.NamesOf([]catalog.Flag{removed}).Free("RENNES", ""))
	require.ErrorIs(t, catalog.NamesOf([]catalog.Flag{ended(t)}).Free("RENNES", ""), catalog.ErrNameTaken,
		"a flag whose time ended keeps its name: a player may claim it")
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

func TestARenamedFlagKeepsItsCodeAndTellsItsNewName(t *testing.T) {
	renamed, fact := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Renamed("Roazhon", closedAt)

	assert.Equal(t, catalog.Name("Roazhon"), renamed.TakeSnapshot().Name)
	assert.Equal(t, catalog.FlagID("r-0000000001"), renamed.TakeSnapshot().ID)
	assert.True(t, proto.Equal(&flagsv1.FlagRenamed{Flag: "r-0000000001", Name: "Roazhon", RenamedAt: timestamppb.New(closedAt)},
		fact), "%v", fact)
}
