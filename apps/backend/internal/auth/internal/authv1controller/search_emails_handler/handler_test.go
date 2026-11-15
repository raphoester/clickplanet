package search_emails_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/search_emails_handler"
)

type search struct {
	text   string
	limit  int
	offset int
}

type stubQuery struct {
	answer *authv1.SearchEmailsResponse
	asked  []search
}

func (s *stubQuery) Holders(_ context.Context, text string, limit int, offset int) (*authv1.SearchEmailsResponse, error) {
	s.asked = append(s.asked, search{text: text, limit: limit, offset: offset})
	return s.answer, nil
}

func found() *stubQuery {
	return &stubQuery{answer: &authv1.SearchEmailsResponse{Holders: []*authv1.EmailHolder{
		{AccountId: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", Email: "ada+1@gmail.com", Provider: "email"},
	}}}
}

func searchEmails(
	t *testing.T, query *stubQuery, req *authv1.SearchEmailsRequest,
) (*connect.Response[authv1.SearchEmailsResponse], error) {
	t.Helper()
	return search_emails_handler.New(query).SearchEmails(t.Context(), connect.NewRequest(req)) //nolint:wrapcheck // the test reads the handler's own error.
}

func TestTheTextIsTrimmedAndTheHoldersAreTheQuerys(t *testing.T) {
	query := found()

	res, err := searchEmails(t, query, &authv1.SearchEmailsRequest{Text: "  +1@ ", Limit: 20, Offset: 40})
	require.NoError(t, err)

	assert.Equal(t, []search{{text: "+1@", limit: 20, offset: 40}}, query.asked)
	assert.True(t, proto.Equal(query.answer, res.Msg))
}

func TestAnEmptyTextIsRefusedAndReadsNothing(t *testing.T) {
	for _, text := range []string{"", "   "} {
		query := found()
		_, err := searchEmails(t, query, &authv1.SearchEmailsRequest{Text: text})
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), text)
		assert.Empty(t, query.asked, text)
	}
}

func TestNoLimitAndTooMuchAreThePageSize(t *testing.T) {
	for asked, want := range map[uint32]int{0: 500, 7: 7, 500: 500, 10_000: 500} {
		query := found()
		_, err := searchEmails(t, query, &authv1.SearchEmailsRequest{Text: "+", Limit: asked})
		require.NoError(t, err)
		assert.Equal(t, []search{{text: "+", limit: want}}, query.asked, asked)
	}
}
