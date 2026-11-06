package nickname_subscriber_test

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

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/nickname_subscriber"
)

var account = links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))

type namings struct {
	named []links.AccountID
	err   error
}

func (n *namings) Execute(_ context.Context, account links.AccountID) error {
	n.named = append(n.named, account)
	return n.err
}

func hear[T nickname_subscriber.Event](t *testing.T, useCase *namings, event T) (string, error) {
	t.Helper()

	var logged bytes.Buffer
	err := nickname_subscriber.New[T](useCase, slog.New(slog.NewTextHandler(&logged, nil))).Handle(t.Context(), event)
	return logged.String(), err
}

func TestEachOfPlayersEventsNamesTheAccount(t *testing.T) {
	useCase := &namings{}

	_, err := hear(t, useCase, &playerv1.PlayerNamed{AccountId: account.String()})
	require.NoError(t, err)
	_, err = hear(t, useCase, &playerv1.PlayerLeaving{AccountId: account.String()})
	require.NoError(t, err)
	_, err = hear(t, useCase, &playerv1.PlayerReturned{AccountId: account.String()})
	require.NoError(t, err)

	assert.Equal(t, []links.AccountID{account, account, account}, useCase.named)
}

func TestANicknameDiscordRefusedIsLoggedAndNotTriedAgain(t *testing.T) {
	logged, err := hear(t, &namings{err: fmt.Errorf("%w: it answered 403", discord.ErrRefused)},
		&playerv1.PlayerNamed{AccountId: account.String()})

	require.NoError(t, err)
	assert.Contains(t, logged, "refused a player's nickname")
	assert.Contains(t, logged, account.String())
}

func TestAFailureThatMayPassIsTriedAgain(t *testing.T) {
	logged, err := hear(t, &namings{err: errors.New("player is down")}, &playerv1.PlayerNamed{AccountId: account.String()})

	require.Error(t, err)
	assert.Empty(t, logged, "log_subscriber logs what the bus tries again")
}

func TestAnEventThatNamesNoAccountIsRefused(t *testing.T) {
	useCase := &namings{}

	_, err := hear(t, useCase, &playerv1.PlayerNamed{AccountId: "nobody"})

	require.ErrorIs(t, err, links.ErrInvalidAccount)
	assert.Empty(t, useCase.named)
}
