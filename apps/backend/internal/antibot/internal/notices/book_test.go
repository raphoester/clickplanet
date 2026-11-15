package notices_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var at = time.Date(2026, 11, 15, 20, 0, 0, 0, time.UTC)

const (
	ada   = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	grace = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a11"
)

func bookOf(store *notices.MemoryStore, feed *notices.MemoryFeed) *notices.Book {
	return notices.NewBook(cptime.NewFixedClock(at), store, feed)
}

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &notices.StoreContractSuite{NewStore: func() notices.Store { return notices.NewMemoryStore() }})
}

func TestAnAccountIsNoticedAndShownIt(t *testing.T) {
	feed := &notices.MemoryFeed{}
	book := bookOf(notices.NewMemoryStore(), feed)

	notice, kept, err := book.Notify(t.Context(), ada, grace)

	require.NoError(t, err)
	require.True(t, kept)
	want := notices.Notice{Account: ada, Alongside: grace, At: at}
	assert.Equal(t, want, notice)
	assert.Equal(t, []notices.Notice{want}, feed.Published())
}

func TestAnAccountIsNoticedOnce(t *testing.T) {
	feed := &notices.MemoryFeed{}
	first := notices.Notice{Account: ada, Alongside: grace, At: at.Add(-24 * time.Hour)}
	book := bookOf(notices.NewMemoryStore(first), feed)

	_, kept, err := book.Notify(t.Context(), ada, grace)

	require.NoError(t, err)
	assert.False(t, kept)
	assert.Empty(t, feed.Published())
	notice, noticed, err := book.Notice(t.Context(), ada)
	require.NoError(t, err)
	require.True(t, noticed)
	assert.Equal(t, first, notice, "the first notice is kept")
}

func TestANoticeIsPendingUntilTheAccountAcknowledgesIt(t *testing.T) {
	book := bookOf(notices.NewMemoryStore(), &notices.MemoryFeed{})
	_, _, err := book.Notify(t.Context(), ada, grace)
	require.NoError(t, err)

	pending, err := book.Pending(t.Context(), ada)
	require.NoError(t, err)
	assert.True(t, pending)

	require.NoError(t, book.Acknowledge(t.Context(), ada))

	pending, err = book.Pending(t.Context(), ada)
	require.NoError(t, err)
	assert.False(t, pending)
}

func TestAnAccountNeverNoticedHasNothingPending(t *testing.T) {
	pending, err := bookOf(notices.NewMemoryStore(), &notices.MemoryFeed{}).Pending(t.Context(), grace)

	require.NoError(t, err)
	assert.False(t, pending)
}

func TestACallerWithNoAccountHasNoNotice(t *testing.T) {
	_, noticed, err := bookOf(notices.NewMemoryStore(), &notices.MemoryFeed{}).Notice(t.Context(), "")

	require.NoError(t, err)
	assert.False(t, noticed)
}

func TestANoticeThatCannotBeKeptSaysSoAndIsNotShown(t *testing.T) {
	store := notices.NewMemoryStore()
	store.FailWith(errors.New("postgres is down"))
	feed := &notices.MemoryFeed{}

	_, _, err := bookOf(store, feed).Notify(t.Context(), ada, grace)

	require.Error(t, err)
	assert.Empty(t, feed.Published())
}
