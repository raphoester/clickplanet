package email_matches_query

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
)

// The addresses are auth's: player keeps none.
type Auth interface {
	SearchEmails(
		ctx context.Context,
		req *connect.Request[authv1.SearchEmailsRequest],
	) (*connect.Response[authv1.SearchEmailsResponse], error)
}

const askTimeout = 5 * time.Second

func New(auth Auth) *Query {
	return &Query{auth: auth}
}

type Query struct {
	auth Auth
}

func (q *Query) Matches(
	ctx context.Context, text string, limit uint32, offset uint32,
) (*playerv1.SearchPlayersByEmailResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := q.auth.SearchEmails(ctx, connect.NewRequest(&authv1.SearchEmailsRequest{
		Text: text, Limit: limit, Offset: offset,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to call auth.v1.InternalService/SearchEmails: %w", err)
	}

	answer := &playerv1.SearchPlayersByEmailResponse{}
	for _, holder := range res.Msg.GetHolders() {
		answer.Matches = append(answer.Matches, &playerv1.EmailMatch{
			AccountId: holder.GetAccountId(),
			Email:     holder.GetEmail(),
			Provider:  holder.GetProvider(),
			LinkedAt:  holder.GetLinkedAt(),
		})
	}
	return answer, nil
}
