package seasons_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
)

var ada = players.AccountID{15: 1}

func TestARecordKeepsEachSeasonPlayedOnceInOrder(t *testing.T) {
	record := seasons.NewRecord(ada).
		With(seasons.Season{Number: 1, Flag: "de"}).
		With(seasons.Season{Number: 0, Flag: "fr", Won: true})

	assert.Equal(t, seasons.RecordSnapshot{Account: ada, Seasons: []seasons.Season{
		{Number: 0, Flag: "fr", Won: true},
		{Number: 1, Flag: "de"},
	}}, record.TakeSnapshot())
}

func TestASeasonToldAgainReplacesWhatWasKept(t *testing.T) {
	record := seasons.NewRecord(ada).
		With(seasons.Season{Number: 0, Flag: "fr"}).
		With(seasons.Season{Number: 0, Flag: "de", Won: true})

	assert.Equal(t, []seasons.Season{{Number: 0, Flag: "de", Won: true}}, record.TakeSnapshot().Seasons)
}

func TestARecordSaysWhichSeasonsWerePlayedAndWon(t *testing.T) {
	record := seasons.RecordSnapshot{Account: ada, Seasons: []seasons.Season{
		{Number: 0, Flag: "fr", Won: true},
		{Number: 1, Flag: "de"},
	}}.Restore()

	assert.True(t, record.Played(0))
	assert.True(t, record.Won(0))
	assert.True(t, record.Played(1))
	assert.False(t, record.Won(1))
	assert.False(t, record.Played(2))
	assert.False(t, seasons.Record{}.Played(0), "an account with no record played nothing")
}

func TestAForgottenRecordHoldsNoSeason(t *testing.T) {
	record := seasons.NewRecord(ada).With(seasons.Season{Number: 0, Flag: "fr"}).Forgotten()

	assert.Equal(t, seasons.RecordSnapshot{Account: ada}, record.TakeSnapshot())
}

func TestARestoredSnapshotKeepsTheFirstOfASeasonListedTwice(t *testing.T) {
	record := seasons.RecordSnapshot{Account: ada, Seasons: []seasons.Season{
		{Number: 0, Flag: "fr"},
		{Number: 0, Flag: "de"},
	}}.Restore()

	assert.Equal(t, []seasons.Season{{Number: 0, Flag: "fr"}}, record.TakeSnapshot().Seasons)
}
