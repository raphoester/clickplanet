package reviews_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/reviews"
)

const smurfs = reviews.SubmissionID("00000000-0000-0000-0000-000000000001")

var at = time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)

func told(t *testing.T, review reviews.Review) pings.Post {
	t.Helper()
	poster := &pings.RecordedPoster{}
	require.NoError(t, reviews.NewDesk(poster, "300", "https://backoffice.clickplanet.lol/").Tell(t.Context(), review))
	require.Len(t, poster.Posts, 1)
	return poster.Posts[0]
}

func TestAnAppealIsToldToTheStaffWithTheJudgesWhyAndALink(t *testing.T) {
	post := told(t, reviews.Review{
		Submission: smurfs, Name: "Smurfland", Cause: reviews.Appeal, Why: "a cartoon's village", AskedAt: at,
	})

	assert.Equal(t, discord.ChannelID("300"), post.Channel)
	assert.Equal(t, discord.Nobody(), post.Audience)
	assert.Equal(t, "🙋 **Smurfland**: its player asks a person to look again at the judge's rejection. "+
		"Judge: “a cartoon's village” https://backoffice.clickplanet.lol/#/submissions/"+string(smurfs), post.Text)
}

func TestAPaidFlagLeftUnsureIsToldToTheStaff(t *testing.T) {
	post := told(t, reviews.Review{
		Submission: smurfs, Name: "Solandia", Cause: reviews.Unsure, Why: "could be fictional", AskedAt: at,
	})

	assert.Equal(t, "🤔 **Solandia** is paid and waits on you: the judge was unsure. "+
		"Judge: “could be fictional” https://backoffice.clickplanet.lol/#/submissions/"+string(smurfs), post.Text)
}

func TestAPlayersNameIsTextNotMarkdown(t *testing.T) {
	post := told(t, reviews.Review{
		Submission: smurfs, Name: "**[x](https://a.test)**", Cause: reviews.Unsure, Why: "line\n# big", AskedAt: at,
	})

	assert.Contains(t, post.Text, `**\*\*\[x\]\(https://a.test\)\*\***`)
	assert.Contains(t, post.Text, `“line \# big”`)
}

func TestACauseTheDeskDoesNotKnowIsAnError(t *testing.T) {
	err := reviews.NewDesk(&pings.RecordedPoster{}, "300", "https://backoffice.clickplanet.lol").
		Tell(t.Context(), reviews.Review{Submission: smurfs, Name: "Solandia", Cause: "bribe"})
	require.ErrorIs(t, err, reviews.ErrUnknownCause)
}
