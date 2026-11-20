package arrivals_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
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
		"a player's link":        {tag: "r", source: arrivals.SourceRecruit, kept: "r"},
		"a player's video":       {tag: "v", source: arrivals.SourceRecruitVideo, kept: "v"},
		"the YouTube link":       {tag: " Y ", source: arrivals.SourceYouTube, kept: "y"},
		"the TikTok address":     {tag: "t", source: arrivals.SourceTikTok, kept: "t"},
		"a player's old link":    {tag: "rc", source: arrivals.SourceRecruit, kept: "rc"},
		"the old YouTube link":   {tag: "YT", source: arrivals.SourceYouTube, kept: "yt"},
		"the CrazyGames page":    {tag: "cg", source: arrivals.SourceCrazyGames, kept: "cg"},
		"a word, not the letter": {tag: "tiktok", source: arrivals.SourceUnknown, kept: "tiktok"},
		"no tag":                 {tag: "", source: arrivals.SourceUnknown, kept: ""},
		"a streamer's link":      {tag: "2poko2", source: arrivals.SourceUnknown, kept: "2poko2"},
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
		"unknown source": func(s arrivals.Snapshot) arrivals.Snapshot { s.Source = "instagram"; return s },
		"no time":        func(s arrivals.Snapshot) arrivals.Snapshot { s.At = time.Time{}; return s },
	} {
		t.Run(name, func(t *testing.T) {
			_, err := broken(whole).Restore()

			assert.ErrorIs(t, err, arrivals.ErrInvalidArrival)
		})
	}
}

func TestOnlyARecruitHasARecruiter(t *testing.T) {
	for _, tag := range []string{"y", "yt", "t", "cg", "", "tiktok"} {
		arrival, err := arrivals.NewArrival(id, ada, arrivals.Landing{Tag: tag, Recruiter: bob}, at)
		require.NoError(t, err)

		_, recruited := arrival.Recruitment(bob)

		assert.False(t, recruited, "%q", tag)
	}
}

func TestARecruitsRecruitmentNamesItsArrivalAndItsRecruiter(t *testing.T) {
	for _, tag := range []string{"r", "v", "rc"} {
		arrival, err := arrivals.NewArrival(id, ada, arrivals.Landing{Tag: tag}, at)
		require.NoError(t, err)

		recruitment, recruited := arrival.Recruitment(bob)

		require.True(t, recruited, "%q", tag)
		assert.Equal(t, arrivals.RecruitmentSnapshot{Arrival: id, Recruiter: bob}, recruitment.TakeSnapshot())
		_, recruited = arrival.Recruitment(arrivals.AccountID{})
		assert.False(t, recruited, "%q: no recruiter, no recruitment", tag)
	}
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
