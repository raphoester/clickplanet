package e2e_test

import (
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
)

func (p *gamer) tryClick(tile uint32, country string) error {
	req := connect.NewRequest(&planetv1.ClickRequest{TileId: tile, CountryId: country})
	p.send(req.Header())
	_, err := p.clicks().Click(p.t.Context(), req)
	return err //nolint:wrapcheck // the test reads the code.
}

func (s *testSuite) TestABanReachesPlanetAndTheCallerIsRefused() {
	t := s.T()
	guest := s.game.newPlayer(t)
	guest.click(4, "is")

	banned, err := antibotv1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL).BanPlayer(t.Context(),
		connect.NewRequest(&antibotv1.BanPlayerRequest{AccountId: guest.account(), Duration: durationpb.New(time.Hour)}))
	require.NoError(t, err)
	require.True(t, banned.Msg.GetEnforced())

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var refused *connect.Error
		if !assert.ErrorAs(c, guest.tryClick(4, "is"), &refused) {
			return
		}
		assert.Equal(c, connect.CodePermissionDenied, refused.Code())
		require.Len(c, refused.Details(), 1)
		detail, err := refused.Details()[0].Value()
		require.NoError(c, err)
		refusal, ok := detail.(*planetv1.BanRefusal)
		require.True(c, ok)
		assert.Equal(c, banned.Msg.GetBannedUntil().AsTime().UnixMilli(), refusal.GetBannedUntilUnixMs())
	}, 10*time.Second, 50*time.Millisecond, "the ban travels through the antibot's outbox to planet's refused callers")

	_, err = planetv1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL).UnbanPlayer(t.Context(),
		connect.NewRequest(&planetv1.UnbanPlayerRequest{AccountId: guest.account()}))
	require.NoError(t, err, "the old path is answered by the antibot until the backoffice moves")

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.NoError(c, guest.tryClick(4, "is"))
	}, 10*time.Second, 50*time.Millisecond, "the unban admits the caller again")
}
