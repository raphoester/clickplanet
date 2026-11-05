package e2e_test

import (
	"bytes"
	"net/http"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1/promov1connect"
)

// Season 0's finale is seeded with the set rush: planet is given it, the operator writes its cards over a background,
// and a page reads the finale's teaser with the values rush plays.
func (s *testSuite) TestAnOperatorWritesASetsCardsAndAPageReadsTheBattlesTeaserFilled() {
	t := s.T()
	ctx := t.Context()
	planet := planetv1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL)
	promo := promov1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL)

	rules, err := planet.GetRules(ctx, connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.NoError(t, err)
	_, err = planet.SaveRuleSet(ctx, connect.NewRequest(&planetv1.SaveRuleSetRequest{
		Set: &planetv1.RuleSet{Name: "rush", Rules: rules.Msg.GetSets()[0].GetRules()},
	}))
	require.NoError(t, err)

	image := []byte("\x89PNG not really")
	upload, err := promo.CreateBackgroundUpload(ctx, connect.NewRequest(&promov1.CreateBackgroundUploadRequest{
		ContentType: "image/png", SizeBytes: uint64(len(image)),
	}))
	require.NoError(t, err)
	put, err := http.NewRequestWithContext(ctx, http.MethodPut, upload.Msg.GetUploadUrl(), bytes.NewReader(image))
	require.NoError(t, err)
	put.Header.Set("Content-Type", "image/png")
	sent, err := http.DefaultClient.Do(put)
	require.NoError(t, err)
	require.NoError(t, sent.Body.Close())
	require.Equal(t, http.StatusOK, sent.StatusCode)
	_, err = promo.KeepBackground(ctx, connect.NewRequest(&promov1.KeepBackgroundRequest{
		BackgroundId: upload.Msg.GetBackgroundId(), Name: "The war",
	}))
	require.NoError(t, err, "the promo module reads the file back from the bucket")

	_, err = promo.SaveCards(ctx, connect.NewRequest(&promov1.SaveCardsRequest{RuleSet: "rush", Cards: []*promov1.CardDraft{{
		BackgroundId: upload.Msg.GetBackgroundId(), Headline: "Clicks refill {clicks.tempo}× faster", Line: "A bank of {clicks.burst}",
	}}}))
	require.NoError(t, err, "the promo module asks planet over the internal listener whether it keeps the set")

	teasers := promov1connect.NewPromoServiceClient(http.DefaultClient, s.game.baseURL, connect.WithHTTPGet())
	teaser, err := teasers.GetTeaser(ctx, connect.NewRequest(&promov1.GetTeaserRequest{BattleId: "season-0/finale"}))
	require.NoError(t, err, "the promo module asks the gamemaster which set the finale plays, and planet what it plays")
	require.Len(t, teaser.Msg.GetCards(), 1)
	card := teaser.Msg.GetCards()[0]
	assert.Equal(t, mediaOrigin+"/promo/backgrounds/"+upload.Msg.GetBackgroundId(), card.GetBackgroundUrl())
	assert.Regexp(t, `^Clicks refill [0-9.]+× faster$`, card.GetHeadline())
	assert.Regexp(t, `^A bank of [0-9]+$`, card.GetLine())
	assert.Equal(t, "public, max-age=60, s-maxage=60", teaser.Header().Get("Cache-Control"))

	_, err = promo.RemoveBackground(ctx, connect.NewRequest(&promov1.RemoveBackgroundRequest{
		BackgroundId: upload.Msg.GetBackgroundId(),
	}))
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "rush's card shows it")
}
