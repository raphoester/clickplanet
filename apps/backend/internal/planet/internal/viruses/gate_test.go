package viruses_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/viruses"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
)

const alice = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

type board struct {
	owners  map[uint32]string
	shields map[uint32]int
}

func (b board) Owner(tile uint32) (string, bool) {
	if tile > 100 {
		return "", false
	}
	return b.owners[tile], true
}

func (b board) Shields(tile uint32) int { return b.shields[tile] }

type flags struct{ err error }

func (f flags) PlayError(context.Context, clicks.Payer, string) error { return f.err }

type reach struct{ err error }

func (r reach) Refuses(string, uint32) error { return r.err }

type refusals struct {
	refusal clicks.Refusal
	refused bool
	err     error
}

func (r refusals) Refusal(context.Context, clicks.Payer) (clicks.Refusal, bool, error) {
	return r.refusal, r.refused, r.err
}

var tiles = board{owners: map[uint32]string{7: "fr", 8: "de", 9: "de"}, shields: map[uint32]int{9: 1}}

func gateWith(allegiance flags, reached reach, refused refusals) viruses.Gate {
	return viruses.NewGate(tiles, cpcountries.New(), allegiance, reached, refused)
}

func aim(g viruses.Gate, t *testing.T, tile uint32, country string) error {
	t.Helper()
	return g.Aim(t.Context(), clicks.Payer{Account: alice}, tile, country) //nolint:wrapcheck // the test reads the gate's own error.
}

func TestAVirusGoesOnAnEnemyTileWithNoShield(t *testing.T) {
	require.NoError(t, aim(gateWith(flags{}, reach{}, refusals{}), t, 8, "fr"))
}

func TestAVirusGoesOnNoTileOfItsFlagNoBlankTileAndNoShieldedTile(t *testing.T) {
	g := gateWith(flags{}, reach{}, refusals{})

	for _, tile := range []uint32{7, 9, 10} {
		require.ErrorIs(t, aim(g, t, tile, "fr"), clicks.ErrNotInfectable, "tile %d", tile)
	}
}

func TestABannedCallerIsRefusedBeforeAnythingElse(t *testing.T) {
	until := time.Date(2026, 11, 16, 13, 0, 0, 0, time.UTC)
	g := gateWith(flags{}, reach{}, refusals{refusal: clicks.NewRefusal(until), refused: true})

	err := aim(g, t, 7, "xx")

	var refusal clicks.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, until, refusal.Until())
}

func TestAFailedReadOfTheRefusalsOrTheFlagRefusesNothing(t *testing.T) {
	unread := fmt.Errorf("%w: down", allegiances.ErrUnread)
	g := gateWith(flags{err: unread}, reach{}, refusals{err: errors.New("down")})

	require.NoError(t, aim(g, t, 8, "fr"))
}

func TestAnUnknownFlagATileOffTheMapAnotherFlagOrATileOutOfReachIsRefused(t *testing.T) {
	notYours := errors.New("you play for de")
	g := gateWith(flags{}, reach{}, refusals{})

	require.ErrorIs(t, aim(g, t, 8, "xx"), clicks.ErrUnknownCountry)
	require.ErrorIs(t, aim(g, t, 101, "fr"), clicks.ErrTileOutOfRange)
	require.ErrorIs(t, aim(gateWith(flags{err: notYours}, reach{}, refusals{}), t, 8, "fr"), notYours)
	require.ErrorIs(t, aim(gateWith(flags{}, reach{err: clicks.ErrOutOfReach}, refusals{}), t, 8, "fr"), clicks.ErrOutOfReach)
}
