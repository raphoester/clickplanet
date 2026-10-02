package playermessage_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
)

func TestEveryTitleOfTheLadderHasAWireValue(t *testing.T) {
	every := players.TitlesOf(players.Stats{TilesTaken: math.MaxUint64, StreakBest: math.MaxUint32})

	encoded := playermessage.Titles(every)

	assert.Len(t, encoded, len(every))
	assert.NotContains(t, encoded, playerv1.Title_TITLE_UNSPECIFIED)
}

func TestTitlesKeepTheirOrderAndAnUnknownOneIsLeftOut(t *testing.T) {
	encoded := playermessage.Titles(players.Titles{players.Settler, "retired", players.Devoted})

	assert.Equal(t, []playerv1.Title{playerv1.Title_TITLE_SETTLER, playerv1.Title_TITLE_DEVOTED}, encoded)
}
