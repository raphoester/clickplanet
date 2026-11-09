package schedule_post_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/usecases/schedule_post_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now  = time.Date(2026, 11, 12, 14, 52, 0, 0, time.UTC)
	slot = calendar.Slot{At: time.Date(2026, 11, 12, 14, 57, 0, 0, time.UTC), Platforms: []calendar.Platform{calendar.TikTok, calendar.YouTube}}
)

// What the service holds, and each post it was sent, in a feed a post joins once made.
type service struct {
	mu        sync.Mutex
	feed      calendar.Feed
	feedError error
	refusal   error
	refusing  map[calendar.Platform]error

	sent       []posts.PostSnapshot
	remembered []calendar.Posting
}

func (s *service) Posted(_ context.Context, since time.Time) (calendar.Feed, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !since.Equal(calendar.PostedSince(now)) {
		return nil, errors.New("asked from the wrong time")
	}
	return s.feed, s.feedError
}

func (s *service) Schedule(_ context.Context, post posts.Post) (posts.Scheduled, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.refusal != nil {
		return posts.Scheduled{}, s.refusal
	}
	s.sent = append(s.sent, post.TakeSnapshot())
	return posts.Scheduled{ID: "post-1", Refused: s.refusing}, nil
}

// The use case remembers what it made, as remembered_feed does: the list itself never shows it here.
func (s *service) Remember(made calendar.Posting) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.remembered = append(s.remembered, made)
	s.feed = append(s.feed, made)
}

func postAt(t *testing.T, slot calendar.Slot) posts.Post {
	t.Helper()

	post, err := posts.NewPost(posts.Draft{
		Slot: slot, Media: "https://media.example/clip-1.mp4", Key: "9b2a", Title: "FRANCE VS TURKEY",
		TikTok: &posts.TikTokWords{Caption: "FRANCE VS TURKEY"}, YouTube: &posts.YouTubeWords{Title: "FRANCE VS TURKEY"},
	})
	require.NoError(t, err)
	return post
}

func useCase(service *service) *schedule_post_usecase.UseCase {
	return schedule_post_usecase.New(service, service, calendar.Default, cptime.NewFixedClock(now))
}

func TestAPostAtAnOpenSlotIsScheduled(t *testing.T) {
	service := &service{feed: calendar.Feed{{At: now.Add(-2 * time.Hour), Platforms: slot.Platforms}}}

	outcome := useCase(service).Execute(t.Context(), postAt(t, slot))

	assert.Equal(t, schedule_post_usecase.Outcome{Scheduled: &posts.Scheduled{ID: "post-1"}}, outcome)
	require.Len(t, service.sent, 1)
	assert.Equal(t, slot.At, service.sent[0].At)
	assert.Equal(t, []calendar.Posting{{ID: "post-1", At: slot.At, Platforms: slot.Platforms}}, service.remembered)
}

func TestAPostIsRememberedOnThePlatformsThatTookItAlone(t *testing.T) {
	service := &service{refusing: map[calendar.Platform]error{calendar.TikTok: errors.New("no connected tiktok account")}}

	useCase(service).Execute(t.Context(), postAt(t, slot))

	assert.Equal(t, []calendar.Posting{{ID: "post-1", At: slot.At, Platforms: []calendar.Platform{calendar.YouTube}}}, service.remembered)
}

func TestAPostAtASlotAnotherPostCrowdsIsTakenAndNothingIsSent(t *testing.T) {
	service := &service{feed: calendar.Feed{{At: slot.At.Add(30 * time.Minute), Platforms: []calendar.Platform{calendar.YouTube}}}}

	outcome := useCase(service).Execute(t.Context(), postAt(t, slot))

	assert.Contains(t, outcome.Taken, "no longer open")
	assert.Nil(t, outcome.Scheduled)
	assert.Empty(t, service.sent)
}

func TestTwoPostsForTheSameSlotAtOnceAreOneScheduledAndOneTaken(t *testing.T) {
	service := &service{}
	schedule := useCase(service)

	outcomes := make([]schedule_post_usecase.Outcome, 2)
	var wg sync.WaitGroup
	for i := range outcomes {
		wg.Go(func() { outcomes[i] = schedule.Execute(t.Context(), postAt(t, slot)) })
	}
	wg.Wait()

	scheduled := 0
	for _, outcome := range outcomes {
		if outcome.Scheduled != nil {
			scheduled++
			continue
		}
		assert.NotEmpty(t, outcome.Taken)
	}
	assert.Equal(t, 1, scheduled)
	assert.Len(t, service.sent, 1)
}

func TestABusyServiceIsBusyAndARefusalIsAFailureThatSaysWhy(t *testing.T) {
	busy := &posts.BusyError{RetryAfter: 90 * time.Second, Reason: "zernio answered 429 to the post: Rate limit exceeded"}

	outcome := useCase(&service{refusal: busy}).Execute(t.Context(), postAt(t, slot))
	assert.Equal(t, schedule_post_usecase.Outcome{Busy: busy}, outcome)

	outcome = useCase(&service{refusal: errors.New("zernio answered 401 to the post: Invalid API key")}).Execute(t.Context(), postAt(t, slot))
	assert.Equal(t, schedule_post_usecase.Outcome{Failed: "zernio answered 401 to the post: Invalid API key"}, outcome)
}

func TestAFeedThatCannotBeReadSendsNothing(t *testing.T) {
	busy := &posts.BusyError{Reason: "zernio could not be reached: connection refused"}
	service := &service{feedError: busy}

	outcome := useCase(service).Execute(t.Context(), postAt(t, slot))
	assert.Equal(t, busy, outcome.Busy)

	service.feedError = errors.New("zernio answered 401 to the list of posts: Invalid API key")
	outcome = useCase(service).Execute(t.Context(), postAt(t, slot))
	assert.Contains(t, outcome.Failed, "failed to read what each platform holds: zernio answered 401")
	assert.Empty(t, service.sent)
}
