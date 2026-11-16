package levels_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/inmemory_level_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type told struct {
	account accounts.AccountID
	kinds   []levels.Kind
	at      time.Time
}

// bonuses keeps what it is told only in a transaction, as the outbox does.
type bonuses struct {
	outbox *cpatomic.FakeOutbox
	told   []told
}

func (b *bonuses) KeepUnlocks(ctx context.Context, account accounts.AccountID, kinds []levels.Kind, at time.Time) error {
	if err := b.outbox.Record(ctx); err != nil {
		return err //nolint:wrapcheck // the test reads it.
	}
	b.told = append(b.told, told{account: account, kinds: kinds, at: at})
	return nil
}

var (
	ada    = accounts.AccountID{15: 1}
	monday = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
)

func setUpBook() (levels.Book, *inmemory_level_store.Store, *bonuses) {
	store, told := inmemory_level_store.New(), &bonuses{outbox: &cpatomic.FakeOutbox{}}
	return levels.NewBook(cpatomic.FakeTransactioner{}, store, told), store, told
}

func TestARaiseTellsTheBonusesEveryKindItsLevelUnlocksInItsTransaction(t *testing.T) {
	book, _, bonuses := setUpBook()

	previous, err := book.Raise(t.Context(), ada, 4, monday)
	require.NoError(t, err)

	assert.Equal(t, levels.Level(0), previous)
	assert.Equal(t, []told{{account: ada, kinds: []levels.Kind{"enclose_clicks"}, at: monday}}, bonuses.told)
}

func TestARaiseThatUnlocksNothingNewStillTellsTheWholeList(t *testing.T) {
	book, _, bonuses := setUpBook()

	_, err := book.Raise(t.Context(), ada, 2, monday)
	require.NoError(t, err)
	_, err = book.Raise(t.Context(), ada, 3, monday.Add(time.Minute))
	require.NoError(t, err)

	require.Len(t, bonuses.told, 2)
	assert.Empty(t, bonuses.told[1].kinds)
}

func TestARaiseAnotherWriteBeatTellsNothing(t *testing.T) {
	book, store, bonuses := setUpBook()
	require.NoError(t, store.Set(t.Context(), levels.ByAccount{ada: 6}, monday))

	previous, err := book.Raise(t.Context(), ada, 5, monday)
	require.NoError(t, err)

	assert.Equal(t, levels.Level(6), previous)
	assert.Empty(t, bonuses.told)
}

func TestARaiseTheStoreRefusesTellsNothing(t *testing.T) {
	book, store, bonuses := setUpBook()
	store.FailWith(errors.New("postgres is gone"))

	_, err := book.Raise(t.Context(), ada, 5, monday)

	require.Error(t, err)
	assert.Empty(t, bonuses.told)
}
