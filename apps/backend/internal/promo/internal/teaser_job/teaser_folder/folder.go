// A run's folder of teasers: a folder a battle, which the job writes promo's teaser.json into for the renderer, and the
// renderer its cards (card-1.jpg, card-2.jpg, …) and its slideshow (teaser.mp4) beside it, for the job to upload.
package teaser_folder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job"
)

const (
	teaserFile = "teaser.json"
	videoFile  = "teaser.mp4"
)

var decoder = protojson.UnmarshalOptions{DiscardUnknown: true}

type Folder string

// The teaser as promo wrote it, in a folder of its own: the folder's name.
func (f Folder) Keep(_ context.Context, due teaser_job.Due) (string, error) {
	name := due.FolderName()
	folder := filepath.Join(string(f), name)
	if err := os.MkdirAll(folder, 0o755); err != nil { //nolint:gosec // a run's files, public data.
		return "", fmt.Errorf("failed to make the folder of the teaser of %s: %w", due.Battle, err)
	}
	if err := os.WriteFile(filepath.Join(folder, teaserFile), due.Teaser, 0o644); err != nil { //nolint:gosec // public data.
		return "", fmt.Errorf("failed to write the teaser of %s: %w", due.Battle, err)
	}
	return name, nil
}

// Every folder the renderer filled, one folder down, by the battle its teaser.json names: the job downloads each render
// into a folder of its own.
func (f Folder) Rendered(_ context.Context) ([]teaser_job.Rendered, error) {
	found, err := filepath.Glob(filepath.Join(string(f), "*", teaserFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read the folder of the teasers: %w", err)
	}
	slices.Sort(found)
	rendered := make([]teaser_job.Rendered, 0, len(found))
	for _, teaserPath := range found {
		folder := filepath.Dir(teaserPath)
		one, err := renderedIn(folder)
		if err != nil {
			return nil, fmt.Errorf("the folder %s: %w", filepath.Base(folder), err)
		}
		rendered = append(rendered, one)
	}
	return rendered, nil
}

func renderedIn(folder string) (teaser_job.Rendered, error) {
	written, err := os.ReadFile(filepath.Join(folder, teaserFile)) //nolint:gosec // a folder of this run.
	if err != nil {
		return teaser_job.Rendered{}, fmt.Errorf("failed to read its teaser: %w", err)
	}
	var teaser promov1.Teaser
	if err := decoder.Unmarshal(written, &teaser); err != nil {
		return teaser_job.Rendered{}, fmt.Errorf("its teaser.json is not a teaser: %w", err)
	}
	if teaser.GetBattleId() == "" {
		return teaser_job.Rendered{}, errors.New("its teaser names no battle")
	}

	video := filepath.Join(folder, videoFile)
	if _, err := os.Stat(video); err != nil {
		return teaser_job.Rendered{}, fmt.Errorf("it holds no %s: %w", videoFile, err)
	}
	photos, err := cardsIn(folder)
	if err != nil {
		return teaser_job.Rendered{}, err
	}
	if len(photos) != len(teaser.GetCards()) {
		return teaser_job.Rendered{}, fmt.Errorf("it holds %d cards of the %d of its teaser", len(photos), len(teaser.GetCards()))
	}
	return teaser_job.Rendered{Battle: teaser_job.BattleID(teaser.GetBattleId()), Photos: photos, Video: teaser_job.File(video)}, nil
}

// card-1.jpg, card-2.jpg, … card-10.jpg, by their number.
func cardsIn(folder string) ([]teaser_job.File, error) {
	found, err := filepath.Glob(filepath.Join(folder, "card-*.jpg"))
	if err != nil {
		return nil, fmt.Errorf("failed to read its cards: %w", err)
	}
	numbered := map[int]teaser_job.File{}
	for _, path := range found {
		number, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(path), "card-"), ".jpg"))
		if err != nil || number < 1 {
			return nil, fmt.Errorf("%s is no card the renderer writes", filepath.Base(path))
		}
		numbered[number] = teaser_job.File(path)
	}
	cards := make([]teaser_job.File, 0, len(numbered))
	for number := 1; number <= len(numbered); number++ {
		card, ok := numbered[number]
		if !ok {
			return nil, fmt.Errorf("its card %d is missing", number)
		}
		cards = append(cards, card)
	}
	return cards, nil
}
