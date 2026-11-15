package listen_for_warnings_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/notices/inprocess_notice_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/listen_for_warnings_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings/inprocess_warning_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	ada   = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	grace = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a11"
)

var at = time.Date(2026, 11, 10, 20, 0, 0, 0, time.UTC)

type enforcing struct{}

func (enforcing) Enforcing() bool { return true }

type recordingSink struct {
	mu         sync.Mutex
	warnings   int
	notices    int
	heartbeats int
}

func (r *recordingSink) SendWarning() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.warnings++
	return nil
}

func (r *recordingSink) SendNotice() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.notices++
	return nil
}

func (r *recordingSink) SendHeartbeat() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.heartbeats++
	return nil
}

func (r *recordingSink) told() (int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.warnings, r.heartbeats
}

func (r *recordingSink) noticed() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.notices
}

type stack struct {
	store       *warnings.MemoryStore
	feed        *inprocess_warning_feed.Feed
	book        *warnings.Book
	noticeStore *notices.MemoryStore
	noticeFeed  *inprocess_notice_feed.Feed
	noticer     *notices.Book
}

func newStack() stack {
	store, feed := warnings.NewMemoryStore(), inprocess_warning_feed.New()
	noticeStore, noticeFeed := notices.NewMemoryStore(), inprocess_notice_feed.New()
	return stack{
		store: store, feed: feed, book: warnings.NewBook(enforcing{}, cptime.NewFixedClock(at), store, feed),
		noticeStore: noticeStore, noticeFeed: noticeFeed,
		noticer: notices.NewBook(cptime.NewFixedClock(at), noticeStore, noticeFeed),
	}
}

func (s stack) usecase() *listen_for_warnings_usecase.UseCase {
	return listen_for_warnings_usecase.New(s.feed, s.book, s.noticeFeed, s.noticer, time.Hour)
}

func (s stack) listen(t *testing.T, sink *recordingSink) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- s.usecase().Execute(ctx, ada, sink) }()

	require.Eventually(t, func() bool { _, beats := sink.told(); return beats > 0 }, time.Second, time.Millisecond)
	return func() {
		cancel()
		require.NoError(t, <-done)
	}
}

func TestAWarningNotAcknowledgedIsToldAsTheStreamOpens(t *testing.T) {
	s := newStack()
	_, err := s.book.Warn(t.Context(), []string{ada})
	require.NoError(t, err)
	sink := &recordingSink{}

	s.listen(t, sink)()

	warned, _ := sink.told()
	assert.Equal(t, 1, warned)
	assert.Zero(t, sink.noticed())
}

func TestAnAcknowledgedWarningIsNotToldAgain(t *testing.T) {
	s := newStack()
	_, err := s.book.Warn(t.Context(), []string{ada})
	require.NoError(t, err)
	require.NoError(t, s.book.Acknowledge(t.Context(), ada))
	sink := &recordingSink{}

	s.listen(t, sink)()

	warned, _ := sink.told()
	assert.Zero(t, warned)
}

func TestAWarningGivenWhileTheStreamIsOpenIsToldAtOnce(t *testing.T) {
	s := newStack()
	sink := &recordingSink{}
	stop := s.listen(t, sink)

	_, err := s.book.Warn(t.Context(), []string{ada})
	require.NoError(t, err)

	require.Eventually(t, func() bool { warned, _ := sink.told(); return warned == 1 }, time.Second, time.Millisecond)
	stop()
}

func TestAWarningThatCannotBeReadEndsTheStream(t *testing.T) {
	s := newStack()
	failed := errors.New("postgres is down")
	s.store.FailWith(failed)

	err := s.usecase().Execute(t.Context(), ada, &recordingSink{})

	require.ErrorIs(t, err, failed)
}

func TestANoticeNotAcknowledgedIsToldAsTheStreamOpens(t *testing.T) {
	s := newStack()
	_, _, err := s.noticer.Notify(t.Context(), ada, grace)
	require.NoError(t, err)
	sink := &recordingSink{}

	s.listen(t, sink)()

	warned, _ := sink.told()
	assert.Zero(t, warned)
	assert.Equal(t, 1, sink.noticed())
}

func TestAnAcknowledgedNoticeIsNotToldAgain(t *testing.T) {
	s := newStack()
	_, _, err := s.noticer.Notify(t.Context(), ada, grace)
	require.NoError(t, err)
	require.NoError(t, s.noticer.Acknowledge(t.Context(), ada))
	sink := &recordingSink{}

	s.listen(t, sink)()

	assert.Zero(t, sink.noticed())
}

func TestANoticeGivenWhileTheStreamIsOpenIsToldAtOnce(t *testing.T) {
	s := newStack()
	sink := &recordingSink{}
	stop := s.listen(t, sink)

	_, _, err := s.noticer.Notify(t.Context(), ada, grace)
	require.NoError(t, err)

	require.Eventually(t, func() bool { return sink.noticed() == 1 }, time.Second, time.Millisecond)
	stop()
}

func TestANoticeThatCannotBeReadEndsTheStream(t *testing.T) {
	s := newStack()
	failed := errors.New("postgres is down")
	s.noticeStore.FailWith(failed)

	err := s.usecase().Execute(t.Context(), ada, &recordingSink{})

	require.ErrorIs(t, err, failed)
}
