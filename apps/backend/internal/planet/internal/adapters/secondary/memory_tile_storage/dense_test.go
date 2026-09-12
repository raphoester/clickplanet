package memory_tile_storage

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain"
	"github.com/stretchr/testify/require"
)

func readBatch(t *testing.T, batch domain.DenseBatch) []string {
	t.Helper()
	require.Len(t, batch.Tiles, len(batch.Tiles)/2*2)

	read := make([]string, 0, len(batch.Tiles)/2)
	for i := 0; i < len(batch.Tiles); i += 2 {
		read = append(read, batch.Codes[binary.LittleEndian.Uint16(batch.Tiles[i:i+2])])
	}

	return read
}

func TestStateBatchDense(t *testing.T) {
	storage := New(9, Config{}, nil)
	require.NoError(t, storage.Set(context.Background(), 2, "fr"))
	require.NoError(t, storage.Set(context.Background(), 4, "gb-eng"))

	dense := func(t *testing.T, start uint32, end uint32) domain.DenseBatch {
		t.Helper()
		batch, err := storage.StateBatchDense(start, end)
		require.NoError(t, err)
		return batch
	}

	t.Run("the whole map, with unowned tiles reading as empty", func(t *testing.T) {
		batch := dense(t, 0, 9)
		require.Equal(t, uint32(0), batch.Start)
		require.Equal(t, []string{"", "", "fr", "", "gb-eng", "", "", "", "", ""}, readBatch(t, batch))
	})

	t.Run("a sub range keeps its offset", func(t *testing.T) {
		batch := dense(t, 4, 5)
		require.Equal(t, uint32(4), batch.Start)
		require.Equal(t, []string{"gb-eng", ""}, readBatch(t, batch))
	})

	t.Run("an end past the last tile is clamped", func(t *testing.T) {
		require.Equal(t, []string{"", ""}, readBatch(t, dense(t, 8, 4_000)))
	})

	t.Run("the code table is a copy, not the live one", func(t *testing.T) {
		batch := dense(t, 0, 0)
		batch.Codes[0] = "tampered"
		require.Equal(t, "", dense(t, 0, 0).Codes[0])
	})

	t.Run("an inverted range is refused", func(t *testing.T) {
		_, err := storage.StateBatchDense(5, 4)
		require.Error(t, err)
	})
}
