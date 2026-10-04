package listen_for_titles_usecase_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inprocess_title_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/listen_for_titles_usecase"
)

var ada = players.AccountID{15: 1}

type recordingSink struct {
	mu     sync.Mutex
	earned []titles.ID
	err    error
}

func (r *recordingSink) SendTitleEarned(standing titles.Standing) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.earned = append(r.earned, standing.Title.ID())
	return r.err
}

func (r *recordingSink) received() []titles.ID {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.earned)
}

func TestOfTheTitlesGrantedAtOnceOnlyTheHighestOfEachTrackIsSent(t *testing.T) {
	feed := inprocess_title_feed.New()
	sink := &recordingSink{}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- listen_for_titles_usecase.New(feed, titles.NewCatalog()).Execute(ctx, ada, sink) }()

	require.Eventually(t, func() bool {
		feed.Publish(ada, titles.IDs{"settler", "raider", "retired", "og", "loyal"})
		return len(sink.received()) > 0
	}, time.Second, 10*time.Millisecond)
	cancel()

	require.NoError(t, <-done)
	assert.Equal(t, []titles.ID{"og", "raider", "loyal"}, sink.received()[:3])
}

func TestASinkFailureEndsTheListening(t *testing.T) {
	feed := inprocess_title_feed.New()
	failed := errors.New("the stream is gone")
	done := make(chan error, 1)
	go func() {
		done <- listen_for_titles_usecase.New(feed, titles.NewCatalog()).Execute(t.Context(), ada, &recordingSink{err: failed})
	}()

	require.Eventually(t, func() bool {
		feed.Publish(ada, titles.IDs{"settler"})
		select {
		case err := <-done:
			return errors.Is(err, failed)
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
}
