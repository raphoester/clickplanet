package cpbootstrap_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/httpbody"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type seasonService struct {
	seasonsv1connect.UnimplementedSeasonServiceHandler
}

func (seasonService) GetFinaleCalendar(
	_ context.Context,
	req *connect.Request[seasonsv1.GetFinaleCalendarRequest],
) (*connect.Response[httpbody.HttpBody], error) {
	if req.Msg.GetNumber() != 3 {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&httpbody.HttpBody{ContentType: "text/plain", Data: []byte("season 3")}), nil
}

func seasonsModule() cpbootstrap.Module {
	return newModule("seasons", func(props cpbootstrap.Props) error {
		return props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
			return seasonsv1connect.NewSeasonServiceHandler(seasonService{}, options...)
		})
	})
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	return res.StatusCode, string(body)
}

func TestARouteTheProtoDeclaresIsServedOnThePublicRouter(t *testing.T) {
	server := cpbootstrap.ServerConfig{BindAddress: freeAddress(t)}

	serveUntil(t, server, func() {
		status, body := get(t, "http://"+server.BindAddress+"/seasons/3/finale.ics")
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, "season 3", body)

		status, _ = get(t, "http://"+server.BindAddress+"/seasons/4/finale.ics")
		assert.Equal(t, http.StatusNotFound, status)

		status, _ = get(t, "http://"+server.BindAddress+"/nothing/here")
		assert.Equal(t, http.StatusNotFound, status)
	}, seasonsModule())
}
