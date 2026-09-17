package antibot_get_map_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase/antibot_get_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubUseCase struct {
	batch clicks.DenseBatch
	err   error
}

func (s stubUseCase) Execute(context.Context, get_map_usecase.In) (clicks.DenseBatch, error) {
	return s.batch, s.err
}

type fakeGuard struct {
	scopes []string
	maps   []float64
	offMap []bool
}

func (g *fakeGuard) Fetched(scope string, maps float64, offMap bool) {
	g.scopes = append(g.scopes, scope)
	g.maps = append(g.maps, maps)
	g.offMap = append(g.offMap, offMap)
}

type board uint32

func (b board) MaxIndex() uint32 { return uint32(b) }

func TestAReadIsReportedAsAShareOfTheMap(t *testing.T) {
	guard := &fakeGuard{}
	batch := clicks.DenseBatch{Start: 1, Codes: []string{"", "fr"}, Tiles: make([]byte, 2*250)}

	got, err := antibot_get_map.New(stubUseCase{batch: batch}, guard, board(1000)).
		Execute(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), get_map_usecase.In{Start: 1, End: 251})

	require.NoError(t, err)
	assert.Equal(t, batch, got, "the batch is the inner one, untouched")
	assert.Equal(t, []string{"2001:db8::/64"}, guard.scopes, "the reader is its scope, as a click is")
	assert.Equal(t, []float64{0.25}, guard.maps)
	assert.Equal(t, []bool{false}, guard.offMap, "1..251 is a span the web app asks for")
}

func TestAReadOutsideTheMapIsReportedAsSuch(t *testing.T) {
	batch := clicks.DenseBatch{Codes: []string{"", "fr"}, Tiles: make([]byte, 2*250)}

	for _, tc := range []struct {
		name   string
		in     get_map_usecase.In
		offMap bool
	}{
		{"the web app's first batch", get_map_usecase.In{Start: 1, End: 10001}, false},
		{"the web app's last batch, clamped", get_map_usecase.In{Start: 250001, End: 257948}, false},
		{"an unset end is the end of the map", get_map_usecase.In{Start: 1}, false},
		{"the bot's walk starts at tile 0", get_map_usecase.In{Start: 0, End: 10000}, true},
		{"the bot's last batch runs past the end", get_map_usecase.In{Start: 250000, End: 257954}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			guard := &fakeGuard{}

			_, err := antibot_get_map.New(stubUseCase{batch: batch}, guard, board(257948)).
				Execute(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), tc.in)

			require.NoError(t, err)
			assert.Equal(t, []bool{tc.offMap}, guard.offMap)
		})
	}
}

func TestARefusedReadReportsNothing(t *testing.T) {
	guard := &fakeGuard{}

	_, err := antibot_get_map.New(stubUseCase{err: clicks.ErrInvalidTileRange}, guard, board(1000)).
		Execute(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), get_map_usecase.In{Start: 5000, End: 6000})

	require.ErrorIs(t, err, clicks.ErrInvalidTileRange, "still the sentinel the handler maps")
	assert.Empty(t, guard.scopes)
}

func TestAnyOtherErrorIsWrapped(t *testing.T) {
	cause := errors.New("disk on fire")

	_, err := antibot_get_map.New(stubUseCase{err: cause}, &fakeGuard{}, board(1000)).
		Execute(t.Context(), get_map_usecase.In{})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, err.Error(), "failed to read the map")
}
