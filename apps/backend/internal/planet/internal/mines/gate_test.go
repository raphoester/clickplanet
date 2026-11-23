package mines_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/mines"
)

const alice = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

type board map[uint32]string

func (b board) Owner(tile uint32) (string, bool) {
	if tile > 100 {
		return "", false
	}
	return b[tile], true
}

type capitals []uint32

func (c capitals) KnocksOut(tile uint32) bool {
	for _, capital := range c {
		if capital == tile {
			return true
		}
	}
	return false
}

type flags struct{ err error }

func (f flags) PlayError(context.Context, clicks.Payer, string) error { return f.err }

type refusals struct {
	refusal clicks.Refusal
	refused bool
	err     error
}

func (r refusals) Refusal(context.Context, clicks.Payer) (clicks.Refusal, bool, error) {
	return r.refusal, r.refused, r.err
}

var tiles = board{7: "fr", 8: "de", 9: "fr"}

func gateWith(allegiance flags, refused refusals) mines.Gate {
	return mines.NewGate(tiles, capitals{9}, catalog.NewOpenCatalog(), allegiance, refused)
}

func lay(g mines.Gate, t *testing.T, tile uint32, country string) error {
	t.Helper()
	return g.Lay(t.Context(), clicks.Payer{Account: alice}, tile, country) //nolint:wrapcheck // the test reads the gate's own error.
}

func TestAMineGoesOnATileOfTheCallersFlag(t *testing.T) {
	require.NoError(t, lay(gateWith(flags{}, refusals{}), t, 7, "fr"))
}

func TestAMineGoesOnNoEnemyTileNoBlankTileAndNoCapitalTheKnockOutPlaysOn(t *testing.T) {
	g := gateWith(flags{}, refusals{})

	for _, tile := range []uint32{8, 10, 9} {
		require.ErrorIs(t, lay(g, t, tile, "fr"), clicks.ErrNotMineable, "tile %d", tile)
	}
}

func TestABannedCallerIsRefusedBeforeAnythingElse(t *testing.T) {
	until := time.Date(2026, 11, 16, 13, 0, 0, 0, time.UTC)
	g := gateWith(flags{}, refusals{refusal: clicks.NewRefusal(until), refused: true})

	var refusal clicks.Refusal
	require.ErrorAs(t, lay(g, t, 7, "xx"), &refusal)
	assert.Equal(t, until, refusal.Until())
}

func TestAFailedReadOfTheRefusalsOrTheFlagRefusesNothing(t *testing.T) {
	unread := fmt.Errorf("%w: down", allegiances.ErrUnread)

	require.NoError(t, lay(gateWith(flags{err: unread}, refusals{err: errors.New("down")}), t, 7, "fr"))
}

func TestAnUnknownFlagATileOffTheMapOrAnotherFlagIsRefused(t *testing.T) {
	notYours := errors.New("you play for de")
	g := gateWith(flags{}, refusals{})

	require.ErrorIs(t, lay(g, t, 7, "xx"), clicks.ErrUnknownCountry)
	require.ErrorIs(t, lay(g, t, 101, "fr"), clicks.ErrTileOutOfRange)
	require.ErrorIs(t, lay(gateWith(flags{err: notYours}, refusals{}), t, 7, "fr"), notYours)
}
