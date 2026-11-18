package pastnames_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

var (
	ada    = players.AccountID{15: 1}
	monday = time.Date(2026, 11, 16, 20, 0, 0, 0, time.UTC)
)

func renamed(t *testing.T, before players.Profile, name players.Name) players.Profile {
	t.Helper()

	after, err := before.Renamed(name, monday)
	require.NoError(t, err)
	return after
}

func TestTheSameNameInAnotherCaseLeavesNone(t *testing.T) {
	for _, name := range []players.Name{"ada_l", "ADA_L", "Ａｄａ_Ｌ", "Ada_L"} {
		before := players.NewProfile(ada, "Ada_L", monday.Add(-time.Hour))

		_, ok := pastnames.PastNameOf(before, renamed(t, before, name), monday)

		assert.False(t, ok, "%q", name)
	}
}
