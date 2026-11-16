package spread_viruses_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/viruses/usecases/spread_viruses_usecase"
)

var noon = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func virusDue(id clicks.VirusID, next time.Time) clicks.Virus {
	return clicks.VirusSnapshot{ID: id, Flag: "fr", Victim: "de", Infected: []uint32{5}, Infections: 1, Next: next}.Restore()
}

type board []clicks.Virus

func (b board) Viruses() []clicks.Virus { return b }

type contagion struct {
	spread []clicks.VirusID
	fails  map[clicks.VirusID]error
}

func (c *contagion) Spread(_ context.Context, id clicks.VirusID, _ int, _ time.Time) (clicks.Virus, error) {
	c.spread = append(c.spread, id)
	return clicks.Virus{}, c.fails[id]
}

type outbreak struct{}

func (outbreak) Most() int                   { return 32 }
func (outbreak) Next() time.Time             { return noon.Add(20 * time.Second) }
func (outbreak) Due(virus clicks.Virus) bool { return virus.Due(noon) }
func open() clicks.Freeze                    { return false }
func frozen() clicks.Freeze                  { return true }
func spreading(c *contagion, freeze func() clicks.Freeze, viruses ...clicks.Virus) *spread_viruses_usecase.UseCase {
	return spread_viruses_usecase.New(board(viruses), c, outbreak{}, freeze)
}

func TestEveryVirusDueSpreadsAndNoOtherOne(t *testing.T) {
	c := &contagion{}

	spread, err := spreading(c, open, virusDue("a", noon), virusDue("b", noon.Add(time.Second)), virusDue("c", noon.Add(-time.Second))).
		Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 2, spread)
	assert.Equal(t, []clicks.VirusID{"a", "c"}, c.spread)
}

func TestNoVirusSpreadsWhileTheMapTakesNoPlay(t *testing.T) {
	c := &contagion{}

	spread, err := spreading(c, frozen, virusDue("a", noon)).Execute(t.Context())

	require.NoError(t, err)
	assert.Zero(t, spread)
	assert.Empty(t, c.spread)
}

func TestAVirusThatFailsLeavesTheOthersToSpread(t *testing.T) {
	down := errors.New("tile out of range")
	c := &contagion{fails: map[clicks.VirusID]error{"a": down, "b": clicks.ErrNoVirus}}

	spread, err := spreading(c, open, virusDue("a", noon), virusDue("b", noon), virusDue("c", noon)).Execute(t.Context())

	require.ErrorIs(t, err, down)
	assert.NotErrorIs(t, err, clicks.ErrNoVirus, "a virus that left the map meanwhile is no failure")
	assert.Equal(t, 1, spread)
	assert.Equal(t, []clicks.VirusID{"a", "b", "c"}, c.spread)
}
