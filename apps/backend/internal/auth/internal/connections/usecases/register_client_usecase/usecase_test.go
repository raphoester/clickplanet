package register_client_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/aes_connection_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/register_client_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const callback = "https://claude.ai/api/mcp/auth_callback"

func TestARegistrationIsTheClientIdAndKeepsNothing(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	sealer, err := aes_connection_sealer.New([]byte("the seed"))
	require.NoError(t, err)
	useCase := register_client_usecase.New(connections.Redirects{callback}, sealer, cptime.NewFixedClock(now))

	out, err := useCase.Execute(t.Context(), register_client_usecase.In{Name: "Claude", RedirectURIs: []string{callback}})
	require.NoError(t, err)
	assert.Equal(t, connections.ClientSnapshot{Name: "Claude", RedirectURIs: []string{callback}, IssuedAt: now}, out.Client)

	opened, err := sealer.OpenedClient(out.ClientID)
	require.NoError(t, err)
	assert.Equal(t, out.Client, opened.TakeSnapshot(), "the id is the registration")

	_, err = useCase.Execute(t.Context(), register_client_usecase.In{Name: "Claude", RedirectURIs: []string{"https://evil.example"}})
	require.ErrorIs(t, err, connections.ErrInvalidRedirect)
}
