package email_matches_query_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/search_players_by_email_handler/email_matches_query"
)

type stubAuth struct {
	answer *authv1.SearchEmailsResponse
	err    error
	asked  []*authv1.SearchEmailsRequest
}

func (s *stubAuth) SearchEmails(
	_ context.Context, req *connect.Request[authv1.SearchEmailsRequest],
) (*connect.Response[authv1.SearchEmailsResponse], error) {
	s.asked = append(s.asked, req.Msg)
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.answer), nil
}

var linkedAt = timestamppb.New(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC))

func TestEachHolderAuthAnswersIsAMatch(t *testing.T) {
	auth := &stubAuth{answer: &authv1.SearchEmailsResponse{Holders: []*authv1.EmailHolder{
		{AccountId: "a", Email: "ada@gmail.com", Provider: "google", LinkedAt: linkedAt},
		{AccountId: "b", Email: "ada+1@gmail.com", Provider: "email", LinkedAt: linkedAt},
	}}}

	answer, err := email_matches_query.New(auth).Matches(t.Context(), "ada", 50, 100)
	require.NoError(t, err)

	assert.True(t, proto.Equal(&authv1.SearchEmailsRequest{Text: "ada", Limit: 50, Offset: 100}, auth.asked[0]))
	assert.True(t, proto.Equal(&playerv1.SearchPlayersByEmailResponse{Matches: []*playerv1.EmailMatch{
		{AccountId: "a", Email: "ada@gmail.com", Provider: "google", LinkedAt: linkedAt},
		{AccountId: "b", Email: "ada+1@gmail.com", Provider: "email", LinkedAt: linkedAt},
	}}, answer), "got %v", answer)
}

func TestAnAuthThatFailsFailsTheRead(t *testing.T) {
	auth := &stubAuth{err: connect.NewError(connect.CodeUnavailable, assert.AnError)}

	_, err := email_matches_query.New(auth).Matches(t.Context(), "ada", 50, 0)
	require.ErrorIs(t, err, assert.AnError)
}
