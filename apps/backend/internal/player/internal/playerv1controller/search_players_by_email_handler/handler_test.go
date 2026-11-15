package search_players_by_email_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/search_players_by_email_handler"
)

type search struct {
	text   string
	limit  uint32
	offset uint32
}

type stubQuery struct {
	answer *playerv1.SearchPlayersByEmailResponse
	asked  []search
}

func (s *stubQuery) Matches(_ context.Context, text string, limit uint32, offset uint32) (*playerv1.SearchPlayersByEmailResponse, error) {
	s.asked = append(s.asked, search{text: text, limit: limit, offset: offset})
	return s.answer, nil
}

func found() *stubQuery {
	return &stubQuery{answer: &playerv1.SearchPlayersByEmailResponse{Matches: []*playerv1.EmailMatch{
		{AccountId: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", Email: "ada+1@gmail.com", Provider: "email"},
	}}}
}

func searchPlayersByEmail(
	t *testing.T, query *stubQuery, req *playerv1.SearchPlayersByEmailRequest,
) (*connect.Response[playerv1.SearchPlayersByEmailResponse], error) {
	t.Helper()
	return search_players_by_email_handler.New(query).SearchPlayersByEmail(t.Context(), connect.NewRequest(req)) //nolint:wrapcheck // the test reads the handler's own error.
}

func TestTheTextIsTrimmedAndTheMatchesAreTheQuerys(t *testing.T) {
	query := found()

	res, err := searchPlayersByEmail(t, query, &playerv1.SearchPlayersByEmailRequest{Text: " +1@ ", Offset: 100})
	require.NoError(t, err)

	assert.Equal(t, []search{{text: "+1@", limit: 50, offset: 100}}, query.asked)
	assert.True(t, proto.Equal(query.answer, res.Msg))
}

func TestAnEmptyTextIsRefusedAndReadsNothing(t *testing.T) {
	for _, text := range []string{"", "   "} {
		query := found()
		_, err := searchPlayersByEmail(t, query, &playerv1.SearchPlayersByEmailRequest{Text: text})
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), text)
		assert.Empty(t, query.asked, text)
	}
}

func TestTheLimitAskedIsCapped(t *testing.T) {
	for asked, want := range map[uint32]uint32{7: 7, 500: 500, 10_000: 500} {
		query := found()
		_, err := searchPlayersByEmail(t, query, &playerv1.SearchPlayersByEmailRequest{Text: "+", Limit: asked})
		require.NoError(t, err)
		assert.Equal(t, []search{{text: "+", limit: want}}, query.asked, asked)
	}
}

func TestTheAnswerIsNeverStored(t *testing.T) {
	res, err := searchPlayersByEmail(t, found(), &playerv1.SearchPlayersByEmailRequest{Text: "+"})
	require.NoError(t, err)
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"), "the answer holds players' addresses")
}
