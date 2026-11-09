package teaser_folder_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job/teaser_folder"
)

func TestATeaserIsKeptAsPromoWroteItInAFolderOfItsOwn(t *testing.T) {
	folder := teaser_folder.Folder(t.TempDir())
	written := []byte(`{"battleId": "season-1/finale"}`)

	name, err := folder.Keep(t.Context(), teaser_job.Due{Battle: "season-1/finale", Teaser: written})

	require.NoError(t, err)
	assert.Equal(t, "season-1_finale", name)
	kept, err := os.ReadFile(filepath.Join(string(folder), name, "teaser.json")) //nolint:gosec // a file the test made.
	require.NoError(t, err)
	assert.Equal(t, written, kept)
}

// A render as the renderer leaves it, in a folder of the run's, for a battle of cards cards.
func render(t *testing.T, root, folder string, battle string, cards int) string {
	t.Helper()
	dir := filepath.Join(root, folder)
	require.NoError(t, os.MkdirAll(dir, 0o750))
	written, err := protojson.Marshal(&promov1.Teaser{BattleId: battle, Cards: make([]*promov1.TeaserCard, cards)})
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
	one := render(t, root, "rendered-season-1_finale", "season-1/finale", 11)
	two := render(t, root, "rendered-entry-7", "entry-7", 1)
	require.NoError(t, os.WriteFile(filepath.Join(root, "stray.txt"), nil, 0o600))

	rendered, err := teaser_folder.Folder(root).Rendered(t.Context())

	require.NoError(t, err)
	require.Len(t, rendered, 2)
	assert.Equal(t, teaser_job.Rendered{
		Battle: "entry-7", Photos: []teaser_job.File{teaser_job.File(filepath.Join(two, "card-1.jpg"))}, Video: teaser_job.File(filepath.Join(two, "teaser.mp4")),
	}, rendered[0])
	assert.Equal(t, teaser_job.BattleID("season-1/finale"), rendered[1].Battle)
	assert.Len(t, rendered[1].Photos, 11)
	assert.Equal(t, teaser_job.File(filepath.Join(one, "card-2.jpg")), rendered[1].Photos[1])
	assert.Equal(t, teaser_job.File(filepath.Join(one, "card-10.jpg")), rendered[1].Photos[9])
}

func TestARenderMissingAFileIsAnErrorThatNamesItsFolder(t *testing.T) {
	for name, spoil := range map[string]func(t *testing.T, dir string){
		"no video": func(t *testing.T, dir string) {
			t.Helper()
			require.NoError(t, os.Remove(filepath.Join(dir, "teaser.mp4")))
		},
		"a card": func(t *testing.T, dir string) {
			t.Helper()
			require.NoError(t, os.Remove(filepath.Join(dir, "card-1.jpg")))
		},
		"a gap": func(t *testing.T, dir string) {
			t.Helper()
			require.NoError(t, os.Rename(filepath.Join(dir, "card-2.jpg"), filepath.Join(dir, "card-3.jpg")))
		},
		"a stray": func(t *testing.T, dir string) {
			t.Helper()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "card-x.jpg"), nil, 0o600))
		},
		"no id": func(t *testing.T, dir string) {
			t.Helper()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "teaser.json"), []byte(`{}`), 0o600))
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			spoil(t, render(t, root, "rendered-entry-7", "entry-7", 2))

			_, err := teaser_folder.Folder(root).Rendered(t.Context())

			require.ErrorContains(t, err, "the folder rendered-entry-7: ")
		})
	}
}

func TestNoRenderIsNone(t *testing.T) {
	rendered, err := teaser_folder.Folder(filepath.Join(t.TempDir(), "missing")).Rendered(t.Context())

	require.NoError(t, err)
	assert.Empty(t, rendered)
}
