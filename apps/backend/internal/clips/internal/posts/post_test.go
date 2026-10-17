package posts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

func TestAPostKeepsTheKeyAndEveryCountryOfItsClip(t *testing.T) {
	written := rendered(1, "VATICAN CITY VS PALAU")
	written.Key = posts.StoryKey{Flag: "va", Continent: "Europe"}
	written.Countries = []posts.TaggedCountry{{Code: "ru", Tag: "russia"}, {Code: "va", Tag: "vaticancity"}, {Code: "pw", Tag: "palau"}}
	vatican, err := written.Restore()
	require.NoError(t, err)

	post := plan.Posts([]posts.Clip{vatican}, time.Date(2026, 10, 17, 1, 0, 0, 0, time.UTC), posts.LoadedDice(0))[0]

	assert.Equal(t, posts.StoryKey{Flag: "va", Continent: "Europe"}, post.TakeSnapshot().Key)
	assert.Equal(t, []string{"ru", "va", "pw"}, post.TakeSnapshot().Countries)
}
