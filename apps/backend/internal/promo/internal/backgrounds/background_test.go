package backgrounds_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
)

var (
	rules = backgrounds.Rules{MaxBytes: 2 << 20}
	at    = time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)
	war   = backgrounds.File{ID: backgrounds.BackgroundID{15: 1}, MediaType: "image/jpeg", Size: 400_000}
)

func TestABackgroundIsAJPEGAPNGOrAWebPUnderTheSize(t *testing.T) {
	for _, media := range []backgrounds.MediaType{"image/jpeg", "image/png", "image/webp"} {
		require.NoError(t, rules.FileError(backgrounds.File{MediaType: media, Size: 2 << 20}), media)
	}
	for _, refused := range []backgrounds.File{
		{MediaType: "image/gif", Size: 100},
		{MediaType: "image/svg+xml", Size: 100},
		{MediaType: "video/mp4", Size: 100},
		{MediaType: "image/png", Size: 0},
		{MediaType: "image/png", Size: 2<<20 + 1},
	} {
		require.ErrorIs(t, rules.FileError(refused), backgrounds.ErrInvalidBackground, "%+v", refused)
	}
}

func TestABackgroundKeptHasAName(t *testing.T) {
	kept, err := backgrounds.Keep(war, "  The war  ", at, rules)

	require.NoError(t, err)
	assert.Equal(t, backgrounds.BackgroundSnapshot{File: war, Name: "The war", KeptAt: at}, kept.TakeSnapshot())
	assert.Equal(t, kept, kept.TakeSnapshot().Restore())

	for _, name := range []string{"", "   ", strings.Repeat("a", 101)} {
		_, err := backgrounds.Keep(war, name, at, rules)
		require.ErrorIs(t, err, backgrounds.ErrInvalidBackground)
	}
}

func TestAnIDIsAUUID(t *testing.T) {
	id, err := backgrounds.BackgroundIDOf("0190f3b1-7c2a-7d3e-8a4b-1c2d3e4f5a6b")
	require.NoError(t, err)
	assert.Equal(t, "0190f3b1-7c2a-7d3e-8a4b-1c2d3e4f5a6b", id.String())

	for _, value := range []string{"", "war", "00000000-0000-0000-0000-000000000000"} {
		_, err := backgrounds.BackgroundIDOf(value)
		require.ErrorIs(t, err, backgrounds.ErrInvalidBackgroundID, value)
	}
}
