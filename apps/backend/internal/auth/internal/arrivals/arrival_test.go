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
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/inmemory_recruitment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var (
	at  = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	ada = arrivals.AccountID{15: 1}
	bob = arrivals.AccountID{15: 2}
	id  = arrivals.ArrivalID{15: 9}
)

func TestTheTagSaysWhereTheAccountCameFrom(t *testing.T) {
	for name, tc := range map[string]struct {
		tag    string
		source arrivals.Source
		kept   string
	}{
		"a player's link":     {tag: "rc", source: arrivals.SourceRecruit, kept: "rc"},
		"the YouTube link":    {tag: " YT ", source: arrivals.SourceYouTube, kept: "yt"},
		"the CrazyGames page": {tag: "cg", source: arrivals.SourceCrazyGames, kept: "cg"},
		"no tag":              {tag: "", source: arrivals.SourceUnknown, kept: ""},
		"a streamer's link":   {tag: "2poko2", source: arrivals.SourceUnknown, kept: "2poko2"},
	} {
		t.Run(name, func(t *testing.T) {
			arrival, err := arrivals.NewArrival(id, ada, arrivals.Landing{Tag: tc.tag}, at)

			require.NoError(t, err)
			assert.Equal(t, arrivals.Snapshot{ID: id, Account: ada, Source: tc.source, Tag: tc.kept, At: at},
				arrival.TakeSnapshot())
		})
	}
}

func TestOnlyAShortPlainWordIsKeptOfTheTag(t *testing.T) {
	for tag, kept := range map[string]string{
		"MrPeter_PT":            "mrpeter_pt",
		" pl-stream ":           "pl-stream",
		"a b":                   "",
		"<script>":              "",
		"émile":                 "",
		strings.Repeat("a", 32): strings.Repeat("a", 32),
		strings.Repeat("a", 33): "",
	} {
		t.Run(tag, func(t *testing.T) {
			arrival, err := arrivals.NewArrival(id, ada, arrivals.Landing{Tag: tag}, at)

			require.NoError(t, err)
			assert.Equal(t, kept, arrival.TakeSnapshot().Tag)
			assert.Equal(t, arrivals.SourceUnknown, arrival.TakeSnapshot().Source)
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

func TestARecruitKeepsItsRecruiterWithItsArrival(t *testing.T) {
	store := inmemory_arrival_store.New()
	recruitments := inmemory_recruitment_store.New()
	register := arrivals.NewRegister(store, recruitments, &arrivals.SequentialIDs{})

	require.NoError(t, register.Record(t.Context(), ada, arrivals.Landing{Tag: "rc", Recruiter: bob}, at))

	assert.Equal(t, []arrivals.Snapshot{{
		ID: arrivals.ArrivalID{0: 0xa, 15: 1}, Account: ada, Source: arrivals.SourceRecruit, Tag: "rc", At: at,
	}}, store.Kept())
	assert.Equal(t, []arrivals.RecruitmentSnapshot{{Arrival: arrivals.ArrivalID{0: 0xa, 15: 1}, Recruiter: bob}}, recruitments.Kept())
}

func TestOnlyARecruitHasARecruiter(t *testing.T) {
	for _, tag := range []string{"yt", "cg", "", "tiktok"} {
		arrival, err := arrivals.NewArrival(id, ada, arrivals.Landing{Tag: tag, Recruiter: bob}, at)
		require.NoError(t, err)

		_, recruited := arrival.Recruitment(bob)

		assert.False(t, recruited, "%q", tag)
	}
}

func TestARecruitsRecruitmentNamesItsArrivalAndItsRecruiter(t *testing.T) {
	arrival, err := arrivals.NewArrival(id, ada, arrivals.Landing{Tag: "rc"}, at)
	require.NoError(t, err)

	recruitment, recruited := arrival.Recruitment(bob)

	require.True(t, recruited)
	assert.Equal(t, arrivals.RecruitmentSnapshot{Arrival: id, Recruiter: bob}, recruitment.TakeSnapshot())
	_, recruited = arrival.Recruitment(arrivals.AccountID{})
	assert.False(t, recruited, "no recruiter, no recruitment")
}

func TestAnAnonymizedRecruitmentKeepsItsArrival(t *testing.T) {
	recruitment, err := arrivals.RecruitmentSnapshot{Arrival: id, Recruiter: bob}.Restore()
	require.NoError(t, err)

	assert.Equal(t, arrivals.RecruitmentSnapshot{Arrival: id}, recruitment.Anonymized().TakeSnapshot())
}

func TestARecruitmentOfNoArrivalIsRefused(t *testing.T) {
	_, err := arrivals.RecruitmentSnapshot{Recruiter: bob}.Restore()

	assert.ErrorIs(t, err, arrivals.ErrInvalidArrival)
}

func TestARecruitmentThatCouldNotBeSavedFailsTheGuest(t *testing.T) {
	recruitments := inmemory_recruitment_store.New()
	recruitments.FailWith(errors.New("postgres is down"))
	lifetime := accounts.Lifetime{}.WithDefaults()
	guests := arrivals.NewGuests(cpatomic.FakeTransactioner{},
		accounts.NewGuests(inmemory_account_store.New(), &accounts.SequentialIDs{}, &accounts.SequentialTokens{}, lifetime),
		arrivals.NewRegister(inmemory_arrival_store.New(), recruitments, &arrivals.SequentialIDs{}))

	_, err := guests.Start(t.Context(), arrivals.Landing{Tag: "rc", Recruiter: bob}, at)

	assert.Error(t, err, "the transaction rolls the guest and its arrival back with it")
}

type failingGuests struct{}

func (failingGuests) Start(context.Context, time.Time) (accounts.Entry, error) {
	return accounts.Entry{}, errors.New("postgres is down")
}

func TestAGuestThatCouldNotStartHasNoArrival(t *testing.T) {
	store := inmemory_arrival_store.New()
	guests := arrivals.NewGuests(cpatomic.FakeTransactioner{}, failingGuests{},
		arrivals.NewRegister(store, inmemory_recruitment_store.New(), &arrivals.SequentialIDs{}))

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
		arrivals.NewRegister(store, inmemory_recruitment_store.New(), &arrivals.SequentialIDs{}))

	_, err := guests.Start(t.Context(), arrivals.Landing{}, at)

	assert.Error(t, err, "the transaction rolls the guest back with it")
}
