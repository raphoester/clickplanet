package arrivals_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/inmemory_arrival_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var (
	at  = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	ada = arrivals.AccountID{15: 1}
	id  = arrivals.ArrivalID{15: 9}
)

func TestTheTagSaysWhereTheAccountCameFrom(t *testing.T) {
	for name, tc := range map[string]struct {
		tag    string
		source arrivals.Source
	}{
		"a player's link":     {tag: "rc", source: arrivals.SourceRecruit},
		"the YouTube link":    {tag: " YT ", source: arrivals.SourceYouTube},
		"no tag":              {tag: "", source: arrivals.SourceUnknown},
		"a tag nobody posted": {tag: "tiktok", source: arrivals.SourceUnknown},
	} {
		t.Run(name, func(t *testing.T) {
			arrival, err := arrivals.NewArrival(id, ada, arrivals.Landing{Tag: tc.tag}, at)

			require.NoError(t, err)
			assert.Equal(t, arrivals.Snapshot{ID: id, Account: ada, Source: tc.source, At: at}, arrival.TakeSnapshot())
		})
	}
}

func TestOnlyAHostIsKeptOfTheReferrer(t *testing.T) {
	for referrer, kept := range map[string]string{
		"www.google.com":                    "www.google.com",
		"Discord.COM":                       "discord.com",
		"":                                  "",
		"https://www.google.com/search?q=a": "",
		"example.com:8080":                  "",
		"exämple.com":                       "",
		strings.Repeat("a", 254):            "",
	} {
		t.Run(referrer, func(t *testing.T) {
			arrival, err := arrivals.NewArrival(id, ada, arrivals.Landing{Referrer: referrer}, at)

			require.NoError(t, err)
			assert.Equal(t, kept, arrival.TakeSnapshot().Referrer)
		})
	}
}

func TestAnArrivalNeedsAnAccount(t *testing.T) {
	_, err := arrivals.NewArrival(id, arrivals.AccountID{}, arrivals.Landing{Tag: "rc"}, at)

	assert.ErrorIs(t, err, arrivals.ErrInvalidArrival)
}

func TestAnAnonymizedArrivalKeepsAllButTheAccount(t *testing.T) {
	arrival, err := arrivals.NewArrival(id, ada, arrivals.Landing{Tag: "rc", Referrer: "discord.com"}, at)
	require.NoError(t, err)

	assert.Equal(t, arrivals.Snapshot{ID: id, Source: arrivals.SourceRecruit, Referrer: "discord.com", At: at},
		arrival.Anonymized().TakeSnapshot())
}

func TestASnapshotRestoresOnlyWhatIsWhole(t *testing.T) {
	whole := arrivals.Snapshot{ID: id, Account: ada, Source: arrivals.SourceUnknown, At: at}
	for name, broken := range map[string]func(arrivals.Snapshot) arrivals.Snapshot{
		"no id":          func(s arrivals.Snapshot) arrivals.Snapshot { s.ID = arrivals.ArrivalID{}; return s },
		"unknown source": func(s arrivals.Snapshot) arrivals.Snapshot { s.Source = "tiktok"; return s },
		"no time":        func(s arrivals.Snapshot) arrivals.Snapshot { s.At = time.Time{}; return s },
	} {
		t.Run(name, func(t *testing.T) {
			_, err := broken(whole).Restore()

			assert.ErrorIs(t, err, arrivals.ErrInvalidArrival)
		})
	}
}

func TestTheRegisterKeepsTheArrivalOfANewAccount(t *testing.T) {
	store := inmemory_arrival_store.New()
	register := arrivals.NewRegister(store, &arrivals.SequentialIDs{})

	require.NoError(t, register.Record(t.Context(), ada, arrivals.Landing{Tag: "rc"}, at))

	assert.Equal(t, []arrivals.Snapshot{{
		ID: arrivals.ArrivalID{0: 0xa, 15: 1}, Account: ada, Source: arrivals.SourceRecruit, At: at,
	}}, store.Kept())
}

type failingGuests struct{}

func (failingGuests) Start(context.Context, time.Time) (accounts.Entry, error) {
	return accounts.Entry{}, errors.New("postgres is down")
}

func TestAGuestStartsWithItsArrival(t *testing.T) {
	store := inmemory_arrival_store.New()
	lifetime := accounts.Lifetime{}.WithDefaults()
	guests := arrivals.NewGuests(cpatomic.FakeTransactioner{},
		accounts.NewGuests(inmemory_account_store.New(), &accounts.SequentialIDs{}, &accounts.SequentialTokens{}, lifetime),
		arrivals.NewRegister(store, &arrivals.SequentialIDs{}))

	entry, err := guests.Start(t.Context(), arrivals.Landing{Tag: "yt"}, at)

	require.NoError(t, err)
	assert.Equal(t, []arrivals.Snapshot{{
		ID: arrivals.ArrivalID{0: 0xa, 15: 1}, Account: entry.Account(), Source: arrivals.SourceYouTube, At: at,
	}}, store.Kept())
}

func TestAGuestThatCouldNotStartHasNoArrival(t *testing.T) {
	store := inmemory_arrival_store.New()
	guests := arrivals.NewGuests(cpatomic.FakeTransactioner{}, failingGuests{},
		arrivals.NewRegister(store, &arrivals.SequentialIDs{}))

	_, err := guests.Start(t.Context(), arrivals.Landing{Tag: "rc"}, at)

	require.Error(t, err)
	assert.Empty(t, store.Kept())
}

func TestAnArrivalThatCouldNotBeSavedFailsTheGuest(t *testing.T) {
	store := inmemory_arrival_store.New()
	store.FailWith(errors.New("postgres is down"))
	lifetime := accounts.Lifetime{}.WithDefaults()
	guests := arrivals.NewGuests(cpatomic.FakeTransactioner{},
		accounts.NewGuests(inmemory_account_store.New(), &accounts.SequentialIDs{}, &accounts.SequentialTokens{}, lifetime),
		arrivals.NewRegister(store, &arrivals.SequentialIDs{}))

	_, err := guests.Start(t.Context(), arrivals.Landing{}, at)

	assert.Error(t, err, "the transaction rolls the guest back with it")
}
