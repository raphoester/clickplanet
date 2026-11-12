// Package page_catalogues brings the game page's catalogues up to their English: its lines, and the names of its
// landmasses, in every language it has a file for.
package page_catalogues

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/catalogue_files"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/landmass_source"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/natural_earth"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/page_source"
)

type Name string

const (
	Lines      Name = "lines"
	Landmasses Name = "landmasses"
)

// One catalogue: its English in order, and where each language's translations go.
type Catalogue struct {
	Name   Name
	Source []translation.Line
	Order  []translation.Key
	Place  catalogue_files.Place
}

type Work struct {
	Catalogues []Catalogue
	Plans      map[Name][]translation.Plan
}

func (w Work) ToSend() int {
	count := 0
	for _, plans := range w.Plans {
		for _, plan := range plans {
			count += len(plan.Send)
		}
	}
	return count
}

// The page's folder, as `npm` sees it.
func Load(ctx context.Context, page string, languages []translation.Language) (Work, error) {
	i18n := filepath.Join(page, "src", "i18n")
	lines := catalogue_files.Place{
		English:      filepath.Join(i18n, "locales", "en.json"),
		Lines:        filepath.Join(i18n, "locales"),
		Fingerprints: filepath.Join(i18n, "fingerprints"),
	}
	found, err := lines.Languages()
	if err != nil {
		return Work{}, fmt.Errorf("failed to list the languages: %w", err)
	}
	if len(languages) == 0 {
		languages = found
	}
	for _, language := range languages {
		if !slices.Contains(found, language) {
			return Work{}, fmt.Errorf("%w: %s (the page has %v)", ErrUnknownLanguage, language, found)
		}
	}
	pageLines, pageOrder, err := page_source.Read(page_source.Files{
		English: lines.English,
		Notes:   filepath.Join(i18n, "notes.json"),
		Code:    filepath.Join(page, "src"),
	})
	if err != nil {
		return Work{}, fmt.Errorf("failed to read the page's lines: %w", err)
	}
	gazetteer, err := natural_earth.Load(ctx, filepath.Join(page, "node_modules", ".cache", "natural-earth", natural_earth.Tag))
	if err != nil {
		return Work{}, fmt.Errorf("failed to read Natural Earth's names: %w", err)
	}
	landmassLines, landmassOrder, err := landmass_source.Read(filepath.Join(page, "static", "landmassNames.json"), gazetteer)
	if err != nil {
		return Work{}, fmt.Errorf("failed to read the landmass names: %w", err)
	}
	work := Work{
		Catalogues: []Catalogue{
			{Name: Lines, Source: pageLines, Order: pageOrder, Place: lines},
			{Name: Landmasses, Source: landmassLines, Order: landmassOrder, Place: catalogue_files.Place{
				Lines:        filepath.Join(i18n, "landmasses"),
				Fingerprints: filepath.Join(i18n, "fingerprints", "landmasses"),
			}},
		},
		Plans: map[Name][]translation.Plan{},
	}
	for _, catalogue := range work.Catalogues {
		for _, language := range languages {
			current, err := catalogue.Place.Catalogue(language)
			if err != nil {
				return Work{}, fmt.Errorf("failed to read the %s %s: %w", language, catalogue.Name, err)
			}
			work.Plans[catalogue.Name] = append(work.Plans[catalogue.Name], translation.PlanOf(language, catalogue.Source, current))
		}
	}
	return work, nil
}

var ErrUnknownLanguage = errors.New("the page has no file for the language")

// A few languages at once: each sends its batches one after the other.
const languagesAtOnce = 4

type Outcome struct {
	Catalogue Name
	translation.Outcome
	Err error
}

// Every plan is translated, a failed language leaving the others; each catalogue is saved as it comes back.
func Translate(ctx context.Context, work Work, translator translation.Translator) []Outcome {
	var (
		group    errgroup.Group
		lock     sync.Mutex
		outcomes []Outcome
	)
	group.SetLimit(languagesAtOnce)
	for _, catalogue := range work.Catalogues {
		for _, plan := range work.Plans[catalogue.Name] {
			group.Go(func() error {
				outcome := translatedOf(ctx, catalogue, plan, translator)
				lock.Lock()
				defer lock.Unlock()
				outcomes = append(outcomes, outcome)
				return nil
			})
		}
	}
	_ = group.Wait() // Each language's error is in its outcome.
	slices.SortFunc(outcomes, func(a, b Outcome) int {
		return cmp.Or(cmp.Compare(a.Catalogue, b.Catalogue), cmp.Compare(a.Plan.Language, b.Plan.Language))
	})
	return outcomes
}

func translatedOf(ctx context.Context, catalogue Catalogue, plan translation.Plan, translator translation.Translator) Outcome {
	outcome, err := translation.Translate(ctx, plan, translator)
	if err != nil {
		return Outcome{Catalogue: catalogue.Name, Outcome: outcome, Err: err}
	}
	if err := catalogue.Place.Save(plan.Language, catalogue.Order, outcome.Catalogue); err != nil {
		return Outcome{Catalogue: catalogue.Name, Outcome: outcome, Err: err}
	}
	return Outcome{Catalogue: catalogue.Name, Outcome: outcome}
}
