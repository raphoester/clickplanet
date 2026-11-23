package feed_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed"
)

var (
	boot   = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	rennes = catalog.Snapshot{ID: "r-0000000001", Name: "Rennes", State: catalog.Open, DecidedAt: boot.Add(time.Minute)}
)

func TestOnlyAFactDecidedSinceTheBootIsNews(t *testing.T) {
	news := feed.NewsSince(boot)

	assert.True(t, news.Fresh(boot))
	assert.True(t, news.Fresh(boot.Add(time.Second)))
	assert.False(t, news.Fresh(boot.Add(-time.Second)), "told again at the boot")
}

func TestAFlagReachesEveryStreamOpen(t *testing.T) {
	board := feed.NewBoard(4)
	first, leaveFirst := board.Attend()
	defer leaveFirst()
	second, leaveSecond := board.Attend()
	defer leaveSecond()

	board.Publish(rennes)

	for _, flags := range []<-chan catalog.Snapshot{first, second} {
		require.Len(t, flags, 1)
		assert.Equal(t, rennes, <-flags)
	}
}

func TestAStreamThatLeftHearsNothing(t *testing.T) {
	board := feed.NewBoard(4)
	flags, leave := board.Attend()
	leave()
	leave()

	board.Publish(rennes)

	_, open := <-flags
	assert.False(t, open)
}

func TestAStreamThatFallsBehindIsClosedNotSkipped(t *testing.T) {
	board := feed.NewBoard(1)
	flags, leave := board.Attend()
	defer leave()

	board.Publish(rennes)
	board.Publish(rennes)

	assert.Equal(t, rennes, <-flags)
	_, open := <-flags
	assert.False(t, open, "its page opens it again and reads the list")
}
