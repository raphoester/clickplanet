package player_forgotten_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/inmemory_graffiti_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/usecases/anonymize_graffiti_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/subscribers/player_forgotten_subscriber"
)

var at = time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)

func painted(id byte, account cpsession.AccountID) graffiti.Snapshot {
	return graffiti.Snapshot{
		ID: graffiti.ID{15: id}, Account: account, Flag: "ro", Kind: graffiti.Swastika, Tiles: []drawings.TileID{1},
		FlaggedAt: at.Add(time.Duration(id) * time.Minute), SeenAt: at.Add(time.Hour), Link: "https://clickplanet.lol/play",
	}
}

func TestAForgottenAccountsGraffitiStayAsNobodys(t *testing.T) {
	ana, bob := cpsession.AccountID{15: 1}, cpsession.AccountID{15: 2}
	store := inmemory_graffiti_storage.New()
	for _, snapshot := range []graffiti.Snapshot{painted(1, ana), painted(2, bob)} {
		kept, err := snapshot.Restore()
		require.NoError(t, err)
		require.NoError(t, store.Save(t.Context(), kept))
	}

	err := player_forgotten_subscriber.New(anonymize_graffiti_usecase.New(store)).Handle(t.Context(),
		&playerv1.PlayerForgotten{AccountId: ana.String()})

	require.NoError(t, err)
	assert.Equal(t, []graffiti.Snapshot{painted(1, cpsession.NoAccount), painted(2, bob)}, store.Kept())
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := player_forgotten_subscriber.New(anonymize_graffiti_usecase.New(inmemory_graffiti_storage.New())).Handle(
		t.Context(), &playerv1.PlayerForgotten{})

	require.ErrorIs(t, err, graffiti.ErrInvalidAccount)
}
