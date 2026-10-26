package get_map_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type mapReader struct {
	batch clicks.DenseBatch
	err   error
}

func (m mapReader) StateBatchDense(uint32, uint32) (clicks.DenseBatch, error) { return m.batch, m.err }

func read(t *testing.T, board clicks.Board, reader mapReader, in get_map_usecase.In) (clicks.DenseBatch, []*planetv1.MapRead, error) {
	t.Helper()
	log := facts.NewLog(cptime.NewFixedClock(time.Date(2026, 10, 26, 20, 0, 0, 0, time.UTC)))

	batch, err := get_map_usecase.New(board, reader, log).Execute(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), in)

	told := log.Take()
	reads := make([]*planetv1.MapRead, 0, len(told))
	for _, fact := range told {
		read, ok := fact.(*planetv1.MapRead)
		require.True(t, ok)
		reads = append(reads, read)
	}
	return batch, reads, err
}

func TestAReadIsToldAsAShareOfTheMap(t *testing.T) {
	batch := clicks.DenseBatch{Start: 1, Codes: []string{"", "fr"}, Tiles: make([]byte, 2*250)}

	got, reads, err := read(t, clicks.NewBoard(1000), mapReader{batch: batch}, get_map_usecase.In{Start: 1, End: 251})

	require.NoError(t, err)
	assert.Equal(t, batch, got)
	require.Len(t, reads, 1)
	assert.Equal(t, "2001:db8::/64", reads[0].GetScope(), "the reader is its scope, as a click is")
	assert.InDelta(t, 0.25, reads[0].GetMaps(), 1e-9)
	assert.False(t, reads[0].GetOffMap(), "1..251 is a span the web app asks for")
}

func TestAReadOutsideTheMapIsToldAsSuch(t *testing.T) {
	batch := clicks.DenseBatch{Codes: []string{"", "fr"}, Tiles: make([]byte, 2*250)}

	for _, tc := range []struct {
		name   string
		in     get_map_usecase.In
		offMap bool
	}{
		{"the web app's first batch", get_map_usecase.In{Start: 1, End: 10001}, false},
		{"the web app's last batch, clamped", get_map_usecase.In{Start: 250001, End: 262119}, false},
		{"an unset end is the end of the map", get_map_usecase.In{Start: 1}, false},
		{"the bot's walk starts at tile 0", get_map_usecase.In{Start: 0, End: 10000}, true},
		{"the bot's last batch runs past the end", get_map_usecase.In{Start: 250000, End: 262125}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, reads, err := read(t, clicks.NewBoard(262119), mapReader{batch: batch}, tc.in)

			require.NoError(t, err)
			require.Len(t, reads, 1)
			assert.Equal(t, tc.offMap, reads[0].GetOffMap())
		})
	}
}

func TestARefusedReadTellsNothing(t *testing.T) {
	_, reads, err := read(t, clicks.NewBoard(1000), mapReader{err: errors.New("out of range")}, get_map_usecase.In{Start: 5000, End: 6000})

	require.ErrorIs(t, err, clicks.ErrInvalidTileRange)
	assert.Empty(t, reads)
}
