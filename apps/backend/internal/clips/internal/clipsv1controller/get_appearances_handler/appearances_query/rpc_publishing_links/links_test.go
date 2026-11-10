package rpc_publishing_links_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_appearances_handler/appearances_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_appearances_handler/appearances_query/rpc_publishing_links"
)

type publishing struct {
	publishingv1connect.UnimplementedInternalServiceHandler

	err   error
	asked []string
}

func (p *publishing) GetPostLinks(
	_ context.Context, req *connect.Request[publishingv1.GetPostLinksRequest],
) (*connect.Response[publishingv1.GetPostLinksResponse], error) {
	p.asked = req.Msg.GetPostIds()
	if p.err != nil {
		return nil, p.err
	}
	return connect.NewResponse(&publishingv1.GetPostLinksResponse{Links: []*publishingv1.PostLink{
		{PostId: "monday", Platform: "tiktok", Url: "https://tiktok.example/7"},
	}}), nil
}

func links(t *testing.T, fake *publishing, logs *bytes.Buffer) rpc_publishing_links.Links {
	t.Helper()
	_, handler := publishingv1connect.NewInternalServiceHandler(fake)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return rpc_publishing_links.New(publishingv1connect.NewInternalServiceClient(server.Client(), server.URL),
		slog.New(slog.NewTextHandler(logs, nil)))
}

func TestEachLinkIsTheAddressPublishingKnows(t *testing.T) {
	fake := &publishing{}

	read, err := links(t, fake, &bytes.Buffer{}).Of(t.Context(), []string{"monday", "today"})

	require.NoError(t, err)
	assert.Equal(t, map[appearances_query.PostOnPlatform]string{{Post: "monday", Platform: "tiktok"}: "https://tiktok.example/7"}, read)
	assert.Equal(t, []string{"monday", "today"}, fake.asked)
}

func TestAPublishingThatDoesNotAnswerLeavesTheLinksOutAndSaysSo(t *testing.T) {
	var logs bytes.Buffer

	read, err := links(t, &publishing{err: connect.NewError(connect.CodeFailedPrecondition, errors.New("publishing is off"))}, &logs).
		Of(t.Context(), []string{"monday"})

	require.NoError(t, err, "a player still reads the posts that name it")
	assert.Empty(t, read)
	assert.Contains(t, logs.String(), "the posts' links are left out")
}
