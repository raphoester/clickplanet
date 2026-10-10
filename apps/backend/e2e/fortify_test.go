package e2e_test

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
)

func landmassesOf(t *testing.T) (map[uint32][]uint32, []string) {
	t.Helper()

	blob, _, err := mapdata.Borders()
	require.NoError(t, err)

	end := 4 + int(binary.LittleEndian.Uint32(blob))
	var header struct {
		Tiles int      `json:"tiles"`
		Codes []string `json:"codes"`
	}
	require.NoError(t, json.Unmarshal(blob[4:end], &header))

	members := map[uint32][]uint32{}
	for i := range header.Tiles {
		landmass := uint32(binary.LittleEndian.Uint16(blob[end+i*2:]))
		members[landmass] = append(members[landmass], uint32(i+1)) //nolint:gosec // a tile id.
	}
	return members, header.Codes
}

func TestTakingAWholeLandmassFortifiesItForEveryScreen(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)

	members, _ := landmassesOf(t)
	var landmass uint32
	for id, tiles := range members {
		if id != 0 && len(tiles) == 3 {
			landmass = id
			break
		}
	}
	require.NotZero(t, landmass, "the map has a landmass of three tiles")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	stream, err := planetv1connect.NewClickServiceClient(http.DefaultClient, game.baseURL).
		ListenForEvents(ctx, connect.NewRequest(&planetv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	require.True(t, stream.Receive(), "the stream opens with a heartbeat")

	fortified := make(chan *planetv1.LandmassFortified, 4)
	go func() {
		for stream.Receive() {
			if event := stream.Msg().GetLandmassFortified(); event != nil {
				fortified <- event
			}
		}
	}()

	for _, tile := range members[landmass] {
		ada.click(tile, "pt")
	}

	select {
	case event := <-fortified:
		assert.Equal(t, landmass, event.GetLandmassId())
		assert.Equal(t, "pt", event.GetCountryId())
		assert.Equal(t, members[landmass][2], event.GetTileId(), "the last tile taken made it whole")
	case <-time.After(5 * time.Second):
		t.Fatal("no landmass_fortified on the stream")
	}

	batch, err := planetv1connect.NewClickServiceClient(http.DefaultClient, game.baseURL).GetMap(t.Context(),
		connect.NewRequest(&planetv1.GetMapRequest{StartTileId: members[landmass][0], EndTileId: members[landmass][2]}))
	require.NoError(t, err)
	shielded := map[uint32]uint32{}
	for _, tile := range batch.Msg.GetShields() {
		shielded[tile.GetTileId()] = tile.GetShields()
	}
	for _, tile := range members[landmass] {
		assert.Equal(t, uint32(1), shielded[tile], "tile %d holds one shield", tile)
	}

	ada.click(members[landmass][0], "es")
	ada.click(members[landmass][0], "es")
	ada.click(members[landmass][0], "pt")
	select {
	case event := <-fortified:
		t.Fatalf("pt fortified again with nobody between: %+v", event)
	case <-time.After(300 * time.Millisecond):
	}
}
