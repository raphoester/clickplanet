package levels_test

import (
	"context"
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

type paid struct {
	account  accounts.AccountID
	level    levels.Level
	unlocked []levels.Kind
}

// bonuses keeps what it is told only in a transaction, as the outbox does.
type bonuses struct {
	outbox *cpatomic.FakeOutbox
	told   []told
	paid   []paid
}

func (b *bonuses) PayLevelUp(ctx context.Context, account accounts.AccountID, level levels.Level, unlocked []levels.Kind) error {
	if err := b.outbox.Record(ctx); err != nil {
		return err //nolint:wrapcheck // the test reads it.
	}
	b.paid = append(b.paid, paid{account: account, level: level, unlocked: unlocked})
	return nil
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

	previous, err := book.Raise(t.Context(), ada, 5, monday)
	require.NoError(t, err)

	assert.Equal(t, levels.Level(0), previous)
	assert.Equal(t, []told{{
		account: ada, kinds: []levels.Kind{"spread_clicks", "shields", "enclose_clicks"}, at: monday,
	}}, bonuses.told)
}

func TestARaiseThatUnlocksNothingNewStillTellsTheWholeList(t *testing.T) {
	book, _, bonuses := setUpBook()

	_, err := book.Raise(t.Context(), ada, 3, monday)
	require.NoError(t, err)
	_, err = book.Raise(t.Context(), ada, 4, monday.Add(time.Minute))
	require.NoError(t, err)

	require.Len(t, bonuses.told, 2)
	assert.Equal(t, []levels.Kind{"spread_clicks", "shields"}, bonuses.told[1].kinds)
}

func TestARaiseAnotherWriteBeatTellsNothing(t *testing.T) {
	book, store, bonuses := setUpBook()
	require.NoError(t, store.RaiseAll(t.Context(), levels.ByAccount{ada: 6}, monday))

	previous, err := book.Raise(t.Context(), ada, 5, monday)
	require.NoError(t, err)

	assert.Equal(t, levels.Level(6), previous)
	assert.Empty(t, bonuses.told)
	assert.Empty(t, bonuses.paid, "a fact told twice raises once, and pays once")
}

func TestARaisePaysOneLevelUpWithWhatTheLevelUnlocks(t *testing.T) {
	book, _, bonuses := setUpBook()

	_, err := book.Raise(t.Context(), ada, 5, monday)
	require.NoError(t, err)

	assert.Equal(t, []paid{{
		account: ada, level: 5, unlocked: []levels.Kind{"spread_clicks", "shields", "enclose_clicks"},
	}}, bonuses.paid)
}

func TestAJumpOfSeveralLevelsIsOneLevelUp(t *testing.T) {
	book, store, bonuses := setUpBook()
	require.NoError(t, store.RaiseAll(t.Context(), levels.ByAccount{ada: 1}, monday))

	_, err := book.Raise(t.Context(), ada, 9, monday)
	require.NoError(t, err)

	require.Len(t, bonuses.paid, 1)
	assert.Equal(t, levels.Level(9), bonuses.paid[0].level)
}

func TestReachingLevelOneIsNoLevelUp(t *testing.T) {
	book, _, bonuses := setUpBook()

	_, err := book.Raise(t.Context(), ada, 1, monday)
	require.NoError(t, err)

	assert.Len(t, bonuses.told, 1, "its unlocks are told all the same")
	assert.Empty(t, bonuses.paid)
}
