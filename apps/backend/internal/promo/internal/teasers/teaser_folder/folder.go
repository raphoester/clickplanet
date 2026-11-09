// A run's folder of teasers: a folder a teaser, which the job writes its teaser.json into for the renderer, and the
// renderer its cards (card-1.jpg, card-2.jpg, …) and its slideshow (teaser.mp4) beside it, for the job to post.
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
	"google.golang.org/protobuf/types/known/timestamppb"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
)

const (
	teaserFile = "teaser.json"
	videoFile  = "teaser.mp4"
)

var decoder = protojson.UnmarshalOptions{DiscardUnknown: true}

type Folder string

// The teaser in a folder of its own, as promo.v1.Teaser in JSON, what the renderer reads: the folder's name.
func (f Folder) Keep(_ context.Context, teaser teasers.Teaser) (string, error) {
	name := teaser.FolderName()
	folder := filepath.Join(string(f), name)
	if err := os.MkdirAll(folder, 0o755); err != nil { //nolint:gosec // a run's files, public data.
		return "", fmt.Errorf("failed to make the folder of the teaser of %s: %w", teaser.Battle, err)
	}
	written, err := protojson.MarshalOptions{Multiline: true}.Marshal(wireOf(teaser))
	if err != nil {
		return "", fmt.Errorf("failed to write the teaser of %s: %w", teaser.Battle, err)
	}
	if err := os.WriteFile(filepath.Join(folder, teaserFile), written, 0o644); err != nil { //nolint:gosec // public data.
		return "", fmt.Errorf("failed to write the teaser of %s: %w", teaser.Battle, err)
	}
	return name, nil
}

// Every folder the renderer filled, by the battle its teaser.json names, at any depth one folder down: the job
// downloads each render into a folder of its own.
func (f Folder) Rendered(_ context.Context) ([]teasers.Rendered, error) {
	found, err := filepath.Glob(filepath.Join(string(f), "*", teaserFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read the folder of the teasers: %w", err)
	}
	slices.Sort(found)
	rendered := make([]teasers.Rendered, 0, len(found))
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

func renderedIn(folder string) (teasers.Rendered, error) {
	written, err := os.ReadFile(filepath.Join(folder, teaserFile)) //nolint:gosec // a folder of this run.
	if err != nil {
		return teasers.Rendered{}, fmt.Errorf("failed to read its teaser: %w", err)
	}
	var teaser promov1.Teaser
	if err := decoder.Unmarshal(written, &teaser); err != nil {
		return teasers.Rendered{}, fmt.Errorf("its teaser.json is not a teaser: %w", err)
	}
	if teaser.GetBattleId() == "" {
		return teasers.Rendered{}, errors.New("its teaser names no battle")
	}

	video := filepath.Join(folder, videoFile)
	if _, err := os.Stat(video); err != nil {
		return teasers.Rendered{}, fmt.Errorf("it holds no %s: %w", videoFile, err)
	}
	photos, err := cardsIn(folder)
	if err != nil {
		return teasers.Rendered{}, err
	}
	if len(photos) != len(teaser.GetCards()) {
		return teasers.Rendered{}, fmt.Errorf("it holds %d cards of the %d of its teaser", len(photos), len(teaser.GetCards()))
	}
	return teasers.Rendered{
		Battle: teasers.BattleID(teaser.GetBattleId()), Media: teasers.Media{Photos: photos, Video: teasers.File(video)},
	}, nil
}

// card-1.jpg, card-2.jpg, … card-10.jpg, by their number.
func cardsIn(folder string) ([]teasers.File, error) {
	found, err := filepath.Glob(filepath.Join(folder, "card-*.jpg"))
	if err != nil {
		return nil, fmt.Errorf("failed to read its cards: %w", err)
	}
	numbered := map[int]teasers.File{}
	for _, path := range found {
		number, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(path), "card-"), ".jpg"))
		if err != nil || number < 1 {
			return nil, fmt.Errorf("%s is no card the renderer writes", filepath.Base(path))
		}
		numbered[number] = teasers.File(path)
	}
	cards := make([]teasers.File, 0, len(numbered))
	for number := 1; number <= len(numbered); number++ {
		card, ok := numbered[number]
		if !ok {
			return nil, fmt.Errorf("its card %d is missing", number)
		}
		cards = append(cards, card)
	}
	return cards, nil
}

func wireOf(teaser teasers.Teaser) *promov1.Teaser {
	wire := &promov1.Teaser{
		BattleId: string(teaser.Battle), StartsAt: timestamppb.New(teaser.StartsAt), EndsAt: timestamppb.New(teaser.EndsAt),
		Description: teaser.Description, Times: teaser.Times, EndsSeason: teaser.EndsSeason,
	}
	for _, card := range teaser.Cards {
		wire.Cards = append(wire.Cards, &promov1.TeaserCard{BackgroundUrl: card.BackgroundURL, Headline: card.Headline, Line: card.Line})
	}
	return wire
}
