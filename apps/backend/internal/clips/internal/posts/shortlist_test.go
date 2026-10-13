package posts_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

func TestTheBestStoriesAreTheFirstOfTheShortlist(t *testing.T) {
	shortlist := posts.ShortlistOf([]string{"KOSOVO IS ATTACKING THE MIDDLE EAST", "FRANCE IS ATTACKING EUROPE", "PORTUGAL STRIKES BACK"})

	best := shortlist.Best(2)

	assert.Equal(t, posts.Shortlist{
		{Pick: 1, Title: "KOSOVO IS ATTACKING THE MIDDLE EAST"},
		{Pick: 2, Title: "FRANCE IS ATTACKING EUROPE"},
	}, best)
	assert.Equal(t, []posts.Pick{1, 2}, best.Picks())
}

func TestAShortlistWithFewerStoriesGivesAllItHas(t *testing.T) {
	assert.Equal(t, []posts.Pick{1}, posts.ShortlistOf([]string{"PORTUGAL STRIKES BACK"}).Best(2).Picks())
}

func TestAQuietHalfDayGivesNoPick(t *testing.T) {
	assert.Empty(t, posts.ShortlistOf(nil).Best(2).Picks())
	assert.Empty(t, posts.ShortlistOf([]string{"PORTUGAL STRIKES BACK"}).Best(0).Picks())
}
