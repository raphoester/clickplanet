package signed_out_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/inmemory_visit_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/forget_visit_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/signed_out_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestASignedOutAccountLeavesTheRoster(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	account, err := players.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	visits := inmemory_visit_storage.New(cptime.NewFixedClock(now))
	visits.Record(presence.Visit{Account: account, Username: "Ada_L", Tag: "aaaaaa", Country: "fr", At: now})

	err = signed_out_subscriber.New(forget_visit_usecase.New(visits)).Handle(t.Context(), &authv1.SignedOut{AccountId: account.String()})

	require.NoError(t, err)
	assert.Empty(t, visits.Visits())
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	visits := inmemory_visit_storage.New(cptime.SystemClock{})

	err := signed_out_subscriber.New(forget_visit_usecase.New(visits)).Handle(t.Context(), &authv1.SignedOut{})

	assert.ErrorIs(t, err, players.ErrInvalidAccount)
}
