package teaser_folder_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/teaser_folder"
)

var start = time.Date(2026, 11, 14, 20, 0, 0, 0, time.UTC)

func finale(cards int) teasers.Teaser {
	teaser := teasers.Teaser{Battle: "season-1/finale", StartsAt: start, EndsAt: start.Add(2 * time.Hour), Times: 5, EndsSeason: true}
	for i := range cards {
		teaser.Cards = append(teaser.Cards, teasers.Card{BackgroundURL: "https://media.test/bg.jpg", Headline: fmt.Sprintf("Rule %d", i+1)})
	}
	return teaser
}

func TestATeaserIsKeptAsPromosTeaserInAFolderOfItsOwn(t *testing.T) {
	folder := teaser_folder.Folder(t.TempDir())

	name, err := folder.Keep(t.Context(), finale(2))

	require.NoError(t, err)
	assert.Equal(t, "season-1_finale", name)
	written, err := os.ReadFile(filepath.Join(string(folder), name, "teaser.json")) //nolint:gosec // a file the test made.
	require.NoError(t, err)
	var teaser promov1.Teaser
	require.NoError(t, protojson.Unmarshal(written, &teaser))
	assert.Equal(t, "season-1/finale", teaser.GetBattleId())
	assert.Equal(t, start, teaser.GetStartsAt().AsTime())
	assert.True(t, teaser.GetEndsSeason())
	assert.Equal(t, uint32(5), teaser.GetTimes())
	assert.Equal(t, []string{"Rule 1", "Rule 2"}, []string{teaser.GetCards()[0].GetHeadline(), teaser.GetCards()[1].GetHeadline()})
	assert.Contains(t, string(written), `"battleId"`, "the renderer reads it in protobuf's JSON")
}

// A render as the renderer leaves it, in a folder of the run's.
func render(t *testing.T, root, folder string, teaser teasers.Teaser, cards int) string {
	t.Helper()
	dir := filepath.Join(root, folder)
	require.NoError(t, os.MkdirAll(dir, 0o750))
	written, err := protojson.Marshal(&promov1.Teaser{BattleId: string(teaser.Battle), Cards: make([]*promov1.TeaserCard, len(teaser.Cards))})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "teaser.json"), written, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "teaser.mp4"), []byte("mp4"), 0o600))
	for i := range cards {
		require.NoError(t, os.WriteFile(filepath.Join(dir, fmt.Sprintf("card-%d.jpg", i+1)), []byte("jpg"), 0o600))
	}
	return dir
}

func TestEachRenderIsReadWithItsCardsByTheirNumber(t *testing.T) {
	root := t.TempDir()
	one := render(t, root, "teaser-season-1_finale", finale(11), 11)
	other := finale(1)
	other.Battle = "entry-7"
	two := render(t, root, "teaser-entry-7", other, 1)
	require.NoError(t, os.WriteFile(filepath.Join(root, "stray.txt"), nil, 0o600))

	rendered, err := teaser_folder.Folder(root).Rendered(t.Context())

	require.NoError(t, err)
	require.Len(t, rendered, 2)
	assert.Equal(t, teasers.Rendered{
		Battle: "entry-7", Media: teasers.Media{Photos: []teasers.File{teasers.File(filepath.Join(two, "card-1.jpg"))}, Video: teasers.File(filepath.Join(two, "teaser.mp4"))},
	}, rendered[0])
	assert.Equal(t, teasers.BattleID("season-1/finale"), rendered[1].Battle)
	assert.Len(t, rendered[1].Media.Photos, 11)
	assert.Equal(t, teasers.File(filepath.Join(one, "card-2.jpg")), rendered[1].Media.Photos[1])
	assert.Equal(t, teasers.File(filepath.Join(one, "card-10.jpg")), rendered[1].Media.Photos[9])
}

func TestARenderMissingAFileIsAnErrorThatNamesItsFolder(t *testing.T) {
	for name, spoil := range map[string]func(dir string){
		"no video": func(dir string) { require.NoError(t, os.Remove(filepath.Join(dir, "teaser.mp4"))) },
		"a card":   func(dir string) { require.NoError(t, os.Remove(filepath.Join(dir, "card-1.jpg"))) },
		"a gap": func(dir string) {
			require.NoError(t, os.Rename(filepath.Join(dir, "card-2.jpg"), filepath.Join(dir, "card-3.jpg")))
		},
		"a stray": func(dir string) { require.NoError(t, os.WriteFile(filepath.Join(dir, "card-x.jpg"), nil, 0o600)) },
		"no id": func(dir string) {
			require.NoError(t, os.WriteFile(filepath.Join(dir, "teaser.json"), []byte(`{}`), 0o600))
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			spoil(render(t, root, "teaser-entry-7", finale(2), 2))

			_, err := teaser_folder.Folder(root).Rendered(t.Context())

			require.ErrorContains(t, err, "the folder teaser-entry-7: ")
		})
	}
}

func TestNoRenderIsNone(t *testing.T) {
	rendered, err := teaser_folder.Folder(filepath.Join(t.TempDir(), "missing")).Rendered(t.Context())

	require.NoError(t, err)
	assert.Empty(t, rendered)
}
