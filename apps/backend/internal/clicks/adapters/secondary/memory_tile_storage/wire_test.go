package memory_tile_storage

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func decodeChunk(t *testing.T, body []byte) (start uint32, tiles []string) {
	t.Helper()

	require.Equal(t, mapChunkMagic, string(body[:4]))
	codeCount := binary.LittleEndian.Uint16(body[4:6])

	codes := make([]string, 0, codeCount)
	cursor := 6
	for range codeCount {
		length := int(body[cursor])
		cursor++
		codes = append(codes, string(body[cursor:cursor+length]))
		cursor += length
	}

	start = binary.LittleEndian.Uint32(body[cursor : cursor+4])
	count := binary.LittleEndian.Uint32(body[cursor+4 : cursor+8])
	cursor += 8

	require.Len(t, body[cursor:], int(count)*2)
	for range count {
		tiles = append(tiles, codes[binary.LittleEndian.Uint16(body[cursor:cursor+2])])
		cursor += 2
	}

	return start, tiles
}

func TestEncodeStateBatch(t *testing.T) {
	storage := New(9, Config{}, nil, nil)
	require.NoError(t, storage.Set(context.Background(), 2, "fr"))
	require.NoError(t, storage.Set(context.Background(), 4, "de"))

	t.Run("the whole map, with unowned tiles reading as empty", func(t *testing.T) {
		start, tiles := decodeChunk(t, mustEncode(t, storage, 0, 9))
		require.Equal(t, uint32(0), start)
		require.Equal(t, []string{"", "", "fr", "", "de", "", "", "", "", ""}, tiles)
	})

	t.Run("a sub range keeps its offset", func(t *testing.T) {
		start, tiles := decodeChunk(t, mustEncode(t, storage, 4, 5))
		require.Equal(t, uint32(4), start)
		require.Equal(t, []string{"de", ""}, tiles)
	})

	t.Run("an end past the last tile is clamped", func(t *testing.T) {
		_, tiles := decodeChunk(t, mustEncode(t, storage, 8, 4_000))
		require.Equal(t, []string{"", ""}, tiles)
	})

	t.Run("an inverted range is refused", func(t *testing.T) {
		_, err := storage.EncodeStateBatch(5, 4)
		require.Error(t, err)
	})
}

func mustEncode(t *testing.T, storage *Storage, start uint32, end uint32) []byte {
	t.Helper()

	body, err := storage.EncodeStateBatch(start, end)
	require.NoError(t, err)

	return body
}
