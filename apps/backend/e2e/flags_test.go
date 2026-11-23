package e2e_test

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1/flagsv1connect"
)

// With no flag opened, a page and the operator read the flags the game ships with.
func (s *testSuite) TestAPageReadsTheFlagsTheGameShipsWith() {
	t := s.T()
	page := flagsv1connect.NewFlagServiceClient(http.DefaultClient, s.game.baseURL, connect.WithHTTPGet())
	operator := flagsv1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL)

	for _, list := range []func() (*connect.Response[flagsv1.ListFlagsResponse], error){
		func() (*connect.Response[flagsv1.ListFlagsResponse], error) {
			return page.ListFlags(t.Context(), connect.NewRequest(&flagsv1.ListFlagsRequest{}))
		},
		func() (*connect.Response[flagsv1.ListFlagsResponse], error) {
			return operator.ListFlags(t.Context(), connect.NewRequest(&flagsv1.ListFlagsRequest{}))
		},
	} {
		listed, err := list()
		require.NoError(t, err)
		require.Len(t, listed.Msg.GetFlags(), 239)
		assert.Equal(t, "af", listed.Msg.GetFlags()[0].GetCode())
		assert.False(t, listed.Msg.GetFlags()[0].GetClosed())
	}
}
