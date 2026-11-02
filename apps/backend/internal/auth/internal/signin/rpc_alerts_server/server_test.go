package rpc_alerts_server_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"

	alertsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/rpc_alerts_server"
)

type alerts struct {
	asked []*alertsv1.JoinServerRequest
	err   error
}

func (a *alerts) JoinServer(
	_ context.Context,
	req *connect.Request[alertsv1.JoinServerRequest],
) (*connect.Response[alertsv1.JoinServerResponse], error) {
	a.asked = append(a.asked, req.Msg)
	if a.err != nil {
		return nil, a.err
	}
	return connect.NewResponse(&alertsv1.JoinServerResponse{}), nil
}

func join(err error) (*alerts, string) {
	var logged bytes.Buffer
	fake := &alerts{err: err}
	rpc_alerts_server.New(fake, slog.New(slog.NewTextHandler(&logged, nil))).
		Join(context.Background(), accounts.AccountID{15: 7}, "4242", "the-token")
	return fake, logged.String()
}

func TestTheAccountTheUserAndTheTokenAreHandedToTheAlerts(t *testing.T) {
	fake, logged := join(nil)

	assert.Len(t, fake.asked, 1)
	assert.Equal(t, accounts.AccountID{15: 7}.String(), fake.asked[0].GetAccountId())
	assert.Equal(t, "4242", fake.asked[0].GetDiscordUserId())
	assert.Equal(t, "the-token", fake.asked[0].GetAccessToken())
	assert.Empty(t, logged)
}

func TestAlertsOffIsNoServerToJoin(t *testing.T) {
	_, logged := join(connect.NewError(connect.CodeUnimplemented, errors.New("404")))

	assert.Empty(t, logged)
}

func TestAJoinThatFailedIsLoggedWithoutTheToken(t *testing.T) {
	_, logged := join(connect.NewError(connect.CodePermissionDenied, errors.New("discord refused")))

	assert.Contains(t, logged, "did not join the server")
	assert.NotContains(t, logged, "the-token")
}
