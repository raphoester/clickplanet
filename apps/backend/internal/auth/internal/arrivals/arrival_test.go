package arrivals_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/inmemory_arrival_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
)

var (
	at  = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	ada = arrivals.AccountID{15: 1}
	id  = arrivals.ArrivalID{15: 9}
)

func TestTheLandingSaysWhereTheAccountCameFrom(t *testing.T) {
	for name, tc := range map[string]struct {
		landing arrivals.Landing
		source  arrivals.Source
		flag    string
	}{
		"a shared link":                    {landing: arrivals.Landing{Flag: "fr"}, source: arrivals.SourceRecruit, flag: "fr"},
		"a shared link typed by hand":      {landing: arrivals.Landing{Flag: " FR "}, source: arrivals.SourceRecruit, flag: "fr"},
		"the YouTube link":                 {landing: arrivals.Landing{Tag: "yt"}, source: arrivals.SourceYouTube},
		"a YouTube link that names a flag": {landing: arrivals.Landing{Flag: "de", Tag: "YT"}, source: arrivals.SourceYouTube, flag: "de"},
		"no query":                         {landing: arrivals.Landing{}, source: arrivals.SourceUnknown},
		"a flag that is no country":        {landing: arrivals.Landing{Flag: "zz"}, source: arrivals.SourceUnknown},
		"a tag nobody posted":              {landing: arrivals.Landing{Flag: "it", Tag: "tiktok"}, source: arrivals.SourceRecruit, flag: "it"},
	} {
		t.Run(name, func(t *testing.T) {
			arrival, err := arrivals.NewArrival(id, ada, tc.landing, cpcountries.New(), at)

			require.NoError(t, err)
			assert.Equal(t, arrivals.Snapshot{ID: id, Account: ada, Source: tc.source, Flag: tc.flag, At: at}, arrival.TakeSnapshot())
		})
	}
}

func TestAnArrivalNeedsAnAccount(t *testing.T) {
	_, err := arrivals.NewArrival(id, arrivals.AccountID{}, arrivals.Landing{Flag: "fr"}, cpcountries.New(), at)

	assert.ErrorIs(t, err, arrivals.ErrInvalidArrival)
}

func TestAnAnonymizedArrivalKeepsAllButTheAccount(t *testing.T) {
	arrival, err := arrivals.NewArrival(id, ada, arrivals.Landing{Flag: "fr"}, cpcountries.New(), at)
	require.NoError(t, err)

	assert.Equal(t, arrivals.Snapshot{ID: id, Source: arrivals.SourceRecruit, Flag: "fr", At: at},
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
	register := arrivals.NewRegister(store, &arrivals.SequentialIDs{}, cpcountries.New())

	require.NoError(t, register.Record(t.Context(), ada, arrivals.Landing{Flag: "pt"}, at))

	assert.Equal(t, []arrivals.Snapshot{{
		ID: arrivals.ArrivalID{0: 0xa, 15: 1}, Account: ada, Source: arrivals.SourceRecruit, Flag: "pt", At: at,
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
		arrivals.NewRegister(store, &arrivals.SequentialIDs{}, cpcountries.New()))

	entry, err := guests.Start(t.Context(), arrivals.Landing{Tag: "yt"}, at)

	require.NoError(t, err)
	assert.Equal(t, []arrivals.Snapshot{{
		ID: arrivals.ArrivalID{0: 0xa, 15: 1}, Account: entry.Account(), Source: arrivals.SourceYouTube, At: at,
	}}, store.Kept())
}

func TestAGuestThatCouldNotStartHasNoArrival(t *testing.T) {
	store := inmemory_arrival_store.New()
	guests := arrivals.NewGuests(cpatomic.FakeTransactioner{}, failingGuests{},
		arrivals.NewRegister(store, &arrivals.SequentialIDs{}, cpcountries.New()))

	_, err := guests.Start(t.Context(), arrivals.Landing{Flag: "fr"}, at)

	require.Error(t, err)
	assert.Empty(t, store.Kept())
}

func TestAnArrivalThatCouldNotBeSavedFailsTheGuest(t *testing.T) {
	store := inmemory_arrival_store.New()
	store.FailWith(errors.New("postgres is down"))
	lifetime := accounts.Lifetime{}.WithDefaults()
	guests := arrivals.NewGuests(cpatomic.FakeTransactioner{},
		accounts.NewGuests(inmemory_account_store.New(), &accounts.SequentialIDs{}, &accounts.SequentialTokens{}, lifetime),
		arrivals.NewRegister(store, &arrivals.SequentialIDs{}, cpcountries.New()))

	_, err := guests.Start(t.Context(), arrivals.Landing{}, at)

	assert.Error(t, err, "the transaction rolls the guest back with it")
}
