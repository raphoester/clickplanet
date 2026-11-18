// Package page_catalogues brings the game page's catalogues up to their English: its lines, the names of each board's
// landmasses, and each flag's country inside a sentence, in every language it has a file for.
package page_catalogues

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/catalogue_files"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/country_source"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/landmass_source"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/natural_earth"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/page_source"
)

type Name string

const (
	Lines     Name = "lines"
	Countries Name = "countries"
)

// A board's landmass names: "landmasses/earth-1".
func LandmassesOf(board string) Name {
	return Name("landmasses/" + board)
}

// The page's own language: its lines are the source, but a country inside a sentence is written for it too.
const English translation.Language = "en"

// One catalogue: its English in order, where each language's translations go, and the languages it is written in.
type Catalogue struct {
	Name      Name
	Source    []translation.Line
	Order     []translation.Key
	Place     catalogue_files.Place
	Languages []translation.Language
}

type Work struct {
	Catalogues    []Catalogue
	Plans         map[Name][]translation.Plan
	MissingLayers []string
}

// The page's folder, as `npm` sees it.
func Load(page string, languages []translation.Language) (Work, error) {
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
		languages = append([]translation.Language{English}, found...)
	}
	for _, language := range languages {
		if language != English && !slices.Contains(found, language) {
			return Work{}, fmt.Errorf("%w: %s (the page has %v)", ErrUnknownLanguage, language, found)
		}
	}
	translated := slices.DeleteFunc(slices.Clone(languages), func(language translation.Language) bool { return language == English })
	pageLines, pageOrder, err := page_source.Read(page_source.Files{
		English: lines.English,
		Notes:   filepath.Join(i18n, "notes.json"),
		Code:    filepath.Join(page, "src"),
	})
	if err != nil {
		return Work{}, fmt.Errorf("failed to read the page's lines: %w", err)
	}
	countryLines, countryOrder, err := country_source.Read(filepath.Join(page, "static", "countries", "countries.json"))
	if err != nil {
		return Work{}, fmt.Errorf("failed to read the flags: %w", err)
	}
	work := Work{Plans: map[Name][]translation.Plan{}}
	pageCatalogue := Catalogue{Name: Lines, Source: pageLines, Order: pageOrder, Place: lines, Languages: translated}
	if err := work.plan(pageCatalogue, func(translation.Language) translation.Memory { return translation.NoMemory{} }); err != nil {
		return Work{}, err
	}
	if err := work.planLandmasses(page, i18n, translated); err != nil {
		return Work{}, err
	}
	countries := Catalogue{
		Name: Countries, Source: countryLines, Order: countryOrder, Languages: languages, Place: catalogue_files.Place{
			Lines:        filepath.Join(i18n, "countries"),
			Fingerprints: filepath.Join(i18n, "fingerprints", "countries"),
		},
	}
	if err := work.plan(countries, func(translation.Language) translation.Memory { return translation.NoMemory{} }); err != nil {
		return Work{}, err
	}
	return work, nil
}

var ErrUnknownLanguage = errors.New("the page has no file for the language")

// Each board's names are a catalogue of their own, but a name one board translates is not planned again on another.
func (w *Work) planLandmasses(page, i18n string, languages []translation.Language) error {
	gazetteer, missing, err := natural_earth.Load(filepath.Join(page, "node_modules", ".cache", "natural-earth", natural_earth.Tag))
	if err != nil {
		return fmt.Errorf("failed to read Natural Earth's names: %w", err)
	}
	w.MissingLayers = missing
	boards, err := landmass_source.Boards(filepath.Join(page, "static", "boards"))
	if err != nil {
		return fmt.Errorf("failed to find the boards' landmass names: %w", err)
	}
	catalogues := make([]Catalogue, 0, len(boards))
	names := map[translation.Language]landmass_source.Names{}
	for _, language := range languages {
		names[language] = landmass_source.Names{}
	}
	for _, board := range boards {
		source, order, err := landmass_source.Read(board.Path, gazetteer)
		if err != nil {
			return fmt.Errorf("failed to read the landmass names of %s: %w", board.Name, err)
		}
		catalogue := Catalogue{
			Name: LandmassesOf(board.Name), Source: source, Order: order, Languages: languages,
			Place: catalogue_files.Place{
				Lines:        filepath.Join(i18n, "landmasses", board.Name),
				Fingerprints: filepath.Join(i18n, "fingerprints", "landmasses", board.Name),
			},
		}
		for _, language := range languages {
			current, err := catalogue.Place.Catalogue(language)
			if err != nil {
				return fmt.Errorf("failed to read the %s %s: %w", language, catalogue.Name, err)
			}
			names[language].Learn(source, current)
		}
		catalogues = append(catalogues, catalogue)
	}
	for _, catalogue := range catalogues {
		if err := w.plan(catalogue, func(language translation.Language) translation.Memory { return names[language] }); err != nil {
			return err
		}
	}
	return nil
}

func (w *Work) plan(catalogue Catalogue, memoryOf func(translation.Language) translation.Memory) error {
	w.Catalogues = append(w.Catalogues, catalogue)
	for _, language := range catalogue.Languages {
		current, err := catalogue.Place.Catalogue(language)
		if err != nil {
			return fmt.Errorf("failed to read the %s %s: %w", language, catalogue.Name, err)
		}
		w.Plans[catalogue.Name] = append(w.Plans[catalogue.Name], translation.PlanOf(language, catalogue.Source, current, memoryOf(language)))
	}
	return nil
}

// Its answers go beside it, named as it is with ".answers.json" for ".json": an object of keys and translations.
type Batch struct {
	Catalogue Name                 `json:"catalogue"`
	Language  translation.Language `json:"language"`
	Lines     []BatchLine          `json:"lines"`
}

type BatchLine struct {
	Key     translation.Key `json:"key"`
	English string          `json:"english"`
	Context string          `json:"context,omitempty"`
}

const answersSuffix = ".answers.json"

var ErrFolderInUse = errors.New("the work folder is not empty: answers from another plan would be taken")

// Writes `<catalogue>/<language>.<n>.json` into the folder, batches as even as the size allows.
func WriteBatches(work Work, folder string, size int) ([]string, error) {
	entries, err := os.ReadDir(folder)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("failed to list %s: %w", folder, err)
	}
	if len(entries) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrFolderInUse, folder)
	}
	var written []string
	for _, catalogue := range work.Catalogues {
		for _, plan := range work.Plans[catalogue.Name] {
			for n, lines := range batchesOf(plan.Send, size) {
				batch := Batch{Catalogue: catalogue.Name, Language: plan.Language, Lines: lines}
				path := filepath.Join(folder, string(catalogue.Name), fmt.Sprintf("%s.%d.json", plan.Language, n+1))
				if err := writeJSON(path, batch); err != nil {
					return nil, err
				}
				written = append(written, path)
			}
		}
	}
	return written, nil
}

func batchesOf(lines []translation.Line, size int) [][]BatchLine {
	if len(lines) == 0 {
		return nil
	}
	count := (len(lines) + size - 1) / size
	batches := make([][]BatchLine, count)
	for i, line := range lines {
		at := i * count / len(lines)
		batches[at] = append(batches[at], BatchLine{Key: line.Key, English: line.English, Context: line.Context})
	}
	return batches
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("failed to make the folder of %s: %w", path, err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

type Outcome struct {
	Catalogue Name
	translation.Outcome
	Err error
}

// Writes every catalogue with the answers found in the folder; a language whose answers do not read is left as it is.
func Apply(work Work, folder string) []Outcome {
	var outcomes []Outcome
	for _, catalogue := range work.Catalogues {
		for _, plan := range work.Plans[catalogue.Name] {
			outcomes = append(outcomes, applied(catalogue, plan, folder))
		}
	}
	return outcomes
}

func applied(catalogue Catalogue, plan translation.Plan, folder string) Outcome {
	answers, err := answersOf(filepath.Join(folder, string(catalogue.Name)), plan.Language)
	if err != nil {
		return Outcome{Catalogue: catalogue.Name, Outcome: translation.Outcome{Plan: plan}, Err: err}
	}
	outcome := translation.Apply(plan, answers)
	if err := catalogue.Place.Save(plan.Language, catalogue.Order, outcome.Catalogue); err != nil {
		return Outcome{Catalogue: catalogue.Name, Outcome: outcome, Err: err}
	}
	return Outcome{Catalogue: catalogue.Name, Outcome: outcome}
}

func answersOf(folder string, language translation.Language) (map[translation.Key]string, error) {
	paths, err := filepath.Glob(filepath.Join(folder, string(language)+".*"+answersSuffix))
	if err != nil {
		return nil, fmt.Errorf("failed to find the answers in %s: %w", folder, err)
	}
	answers := map[translation.Key]string{}
	for _, path := range paths {
		keyed, err := catalogue_files.Read(path)
		if err != nil {
			return nil, err //nolint:wrapcheck // it names the file.
		}
		for key, text := range keyed.Lines {
			answers[key] = text
		}
	}
	return answers, nil
}
