package activity_test

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const account = "00000000-0000-4000-8000-000000000001"

func TestACallerCarriesTheAccountItsTokenNames(t *testing.T) {
	caller := activity.CallerOf(clicks.Payer{Scope: "2001:db8::/64", Account: account, Linked: true})

	assert.Equal(t, activity.Caller{
		Scope:    "2001:db8::/64",
		Account:  cpsession.AccountID(uuid.MustParse(account)),
		SignedIn: true,
	}, caller)
}

func TestACallerWithNoAccountHasNone(t *testing.T) {
	assert.Equal(t, activity.Caller{Scope: "2001:db8::/64", Account: cpsession.NoAccount},
		activity.CallerOf(clicks.Payer{Scope: "2001:db8::/64"}))
	assert.Equal(t, cpsession.NoAccount, activity.CallerOf(clicks.Payer{Account: "not-a-uuid"}).Account)
}

func TestABoxCallerIsItsScopeAlone(t *testing.T) {
	assert.Equal(t, activity.Caller{Scope: "2001:db8::/64", Account: cpsession.NoAccount},
		activity.CallerOfScope("2001:db8::/64"))
}

func TestTrimmedStripsWhatPostgresWouldRefuse(t *testing.T) {
	event := activity.Event{
		Caller:  activity.Caller{Scope: "2001:db8::\x00/64"},
		Country: "f\xffr",
		Held:    "d\x00e",
	}.Trimmed()

	assert.Equal(t, "2001:db8::/64", event.Caller.Scope, "a NUL fails a whole COPY")
	assert.Equal(t, "f�r", event.Country, "so does invalid UTF-8")
	assert.Equal(t, "de", event.Held)
}

func TestTrimmedBoundsAStringAtARuneBoundary(t *testing.T) {
	event := activity.Event{Country: strings.Repeat("é", 100)}.Trimmed()

	assert.LessOrEqual(t, len(event.Country), 64)
	assert.True(t, utf8.ValidString(event.Country), "a cut in the middle of a rune is invalid UTF-8 again")
	assert.Equal(t, strings.Repeat("é", 32), event.Country)
}

func TestTrimmedLeavesAnOrdinaryEventAlone(t *testing.T) {
	event := activity.Event{
		At: time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC), Kind: activity.KindTake,
		Caller: activity.Caller{Scope: "2001:db8::/64"}, Tile: 42, Country: "bg", Held: "fr",
	}

	assert.Equal(t, event, event.Trimmed())
}

func TestDiscardRecordsNothingAndDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() { activity.Discard{}.Record(activity.Event{Kind: activity.KindClick}) })
}

func TestAConfigLeftAtZeroTakesTheDefaults(t *testing.T) {
	config := activity.Config{}.WithDefaults()

	assert.Equal(t, time.Second, config.FlushInterval)
	assert.Equal(t, 100_000, config.MaxPending)
	assert.Equal(t, 72*time.Hour, config.Retention, "what the privacy policy promises")
	assert.Equal(t, 5*time.Minute, config.SweepInterval)
	assert.Equal(t, 10_000_000, config.MaxEvents)
}

func TestTheRecorderOffNeedsNoDatabase(t *testing.T) {
	require.NoError(t, activity.Config{}.Validate())
}

func TestTheRecorderOnNeedsItsDatabase(t *testing.T) {
	require.ErrorContains(t, activity.Config{Enabled: true}.Validate(), "activity.database")
}

func TestANegativeBoundIsRefused(t *testing.T) {
	config := activity.Config{Enabled: true, Database: database(), Retention: -time.Hour}

	require.ErrorContains(t, config.Validate(), "may not be negative")
}

func TestACompleteConfigIsValid(t *testing.T) {
	require.NoError(t, activity.Config{Enabled: true, Database: database()}.Validate())
}

func database() cppg.Config {
	return cppg.Config{
		Host: "localhost", Port: "5432", User: "postgres", DBName: "postgres", SSLMode: "disable", Schema: "activity",
	}
}
