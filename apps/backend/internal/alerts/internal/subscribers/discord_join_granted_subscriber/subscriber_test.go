package discord_join_granted_subscriber_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/join_server_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/discord_join_granted_subscriber"
)

var (
	account = links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))
	event   = &authv1.DiscordJoinGranted{GrantId: "the-grant", AccountId: account.String(), DiscordUserId: "4242"}
	granted = links.Grant{Account: account, User: "4242", Token: "the-access-token"}
)

type stubGrants struct {
	asked []links.GrantID
	err   error
}

func (g *stubGrants) Grant(_ context.Context, id links.GrantID) (links.Grant, error) {
	g.asked = append(g.asked, id)
	if g.err != nil {
		return links.Grant{}, g.err
	}
	return granted, nil
}

type joins struct {
	joined []join_server_usecase.In
	err    error
}

func (j *joins) Execute(_ context.Context, in join_server_usecase.In) error {
	j.joined = append(j.joined, in)
	return j.err
}

func hear(t *testing.T, grants *stubGrants, useCase *joins) (string, error) {
	t.Helper()

	var logged bytes.Buffer
	err := discord_join_granted_subscriber.New(grants, useCase, slog.New(slog.NewTextHandler(&logged, nil))).
		Handle(t.Context(), event)
	return logged.String(), err
}

func TestAGrantJoinsTheServerWithWhatAuthKept(t *testing.T) {
	grants, useCase := &stubGrants{}, &joins{}

	logged, err := hear(t, grants, useCase)

	require.NoError(t, err)
	assert.Equal(t, []links.GrantID{"the-grant"}, grants.asked)
	assert.Equal(t, []join_server_usecase.In{{Account: account, User: "4242", Token: "the-access-token"}}, useCase.joined)
	assert.Empty(t, logged)
}

func TestAGrantAuthKeepsNoMoreIsLoggedAndNotTriedAgain(t *testing.T) {
	useCase := &joins{}

	logged, err := hear(t, &stubGrants{err: fmt.Errorf("%w: the-grant", links.ErrNoGrant)}, useCase)

	require.NoError(t, err, "nothing to join with: the bus may forget it")
	assert.Empty(t, useCase.joined)
	assert.Contains(t, logged, "grant was gone")
	assert.Contains(t, logged, account.String())
}

func TestATokenDiscordRefusedIsLoggedWithoutItAndNotTriedAgain(t *testing.T) {
	logged, err := hear(t, &stubGrants{}, &joins{err: fmt.Errorf("%w: it answered 403", discord.ErrRefused)})

	require.NoError(t, err)
	assert.Contains(t, logged, "did not join the server")
	assert.NotContains(t, logged, "the-access-token")
}

func TestAFailureThatMayPassIsTriedAgain(t *testing.T) {
	for name, tc := range map[string]struct {
		grants  *stubGrants
		useCase *joins
	}{
		"auth unreachable":   {grants: &stubGrants{err: errors.New("auth is unreachable")}, useCase: &joins{}},
		"Discord busy":       {grants: &stubGrants{}, useCase: &joins{err: errors.New("discord answered 429")}},
		"Discord down":       {grants: &stubGrants{}, useCase: &joins{err: errors.New("discord answered 502")}},
		"its own store down": {grants: &stubGrants{}, useCase: &joins{err: errors.New("postgres is down")}},
	} {
		t.Run(name, func(t *testing.T) {
			logged, err := hear(t, tc.grants, tc.useCase)

			require.Error(t, err)
			assert.Empty(t, logged, "log_subscriber logs what the bus tries again")
		})
	}
}
