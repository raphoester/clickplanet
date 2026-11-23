package email_query

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Auth interface {
	GetEmail(ctx context.Context, req *connect.Request[authv1.GetEmailRequest]) (*connect.Response[authv1.GetEmailResponse], error)
}

const askTimeout = 5 * time.Second

func New(auth Auth) *Query {
	return &Query{auth: auth}
}

type Query struct {
	auth Auth
}

func (q *Query) Email(ctx context.Context, account players.AccountID) (*playerv1.GetEmailResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := q.auth.GetEmail(ctx, connect.NewRequest(&authv1.GetEmailRequest{AccountId: account.String()}))
	if err != nil {
		return nil, fmt.Errorf("failed to call auth.v1.InternalService/GetEmail: %w", err)
	}
	return &playerv1.GetEmailResponse{Email: res.Msg.Email}, nil
}
