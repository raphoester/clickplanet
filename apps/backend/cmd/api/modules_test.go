package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var protoPackage = regexp.MustCompile(`/generated/proto/([a-z]+)/v1(?:/|$)`)

// cpflags hears the flags module's facts for the module that keeps a catalog.
var sharedPackage = regexp.MustCompile(`/internal/shared/(cpflags)$`)

var sharedOwners = map[string]string{"cpflags": "flags"}

// A module depends on another when its code speaks the other's proto package: it calls it, hears its events or sends
// it commands, itself or through a shared package. The deprecated mint's package is auth's.
func moduleEdges(t *testing.T) map[string][]string {
	t.Helper()

	owners := map[string]string{"session": "auth"}
	modules, err := os.ReadDir("../../internal")
	require.NoError(t, err)

	edges := map[string][]string{}
	for _, module := range modules {
		if !module.IsDir() || module.Name() == "shared" {
			continue
		}
		name := module.Name()
		edges[name] = nil
		require.NoError(t, filepath.WalkDir(filepath.Join("../../internal", name), func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err //nolint:wrapcheck // the test reads it.
			}
			for _, imported := range file.Imports {
				importPath := strings.Trim(imported.Path.Value, `"`)
				var owner string
				if match := protoPackage.FindStringSubmatch(importPath); match != nil {
					owner = match[1]
				} else if match := sharedPackage.FindStringSubmatch(importPath); match != nil {
					owner = sharedOwners[match[1]]
				} else {
					continue
				}
				if alias, ok := owners[owner]; ok {
					owner = alias
				}
				if owner != name && !slices.Contains(edges[name], owner) {
					edges[name] = append(edges[name], owner)
				}
			}
			return nil
		}))
	}
	return edges
}

func TestTheModulesDependOnEachOtherWithNoCycle(t *testing.T) {
	edges := moduleEdges(t)

	const (
		unseen = iota
		open
		done
	)
	state := map[string]int{}
	var path []string
	var visit func(module string)
	visit = func(module string) {
		switch state[module] {
		case open:
			assert.Fail(t, "a cycle between modules", "%s -> %s", strings.Join(path, " -> "), module)
			return
		case done:
			return
		}
		state[module] = open
		path = append(path, module)
		for _, next := range edges[module] {
			visit(next)
		}
		path = path[:len(path)-1]
		state[module] = done
	}
	for module := range edges {
		visit(module)
	}
}

func TestNothingButTheBrowserAndTheOperatorTalksToProgression(t *testing.T) {
	for module, upstream := range moduleEdges(t) {
		assert.NotContains(t, upstream, "progression", "%s speaks progression's language", module)
	}
}

// Progression tells the bonuses module what each account unlocked, in the bonuses' words, as commands: the gamemaster
// plays the bonuses' rule sets and progression hears the gamemaster, so progression is downstream of the bonuses.
func TestProgressionTellsTheBonusesWhatEachAccountUnlocked(t *testing.T) {
	edges := moduleEdges(t)

	assert.Contains(t, edges["progression"], "bonuses")
	assert.NotContains(t, edges["bonuses"], "progression", "the bonuses module knows no level")
}

func TestOnlyPlayerAlertsAndTheConnectorSpeakAuthsLanguage(t *testing.T) {
	var speakers []string
	for module, upstream := range moduleEdges(t) {
		if slices.Contains(upstream, "auth") {
			speakers = append(speakers, module)
		}
	}
	slices.Sort(speakers)

	assert.Equal(t, []string{"alerts", "mcp", "player"}, speakers)
}

func TestProgressionKnowsPlayerAndNotAuth(t *testing.T) {
	upstream := moduleEdges(t)["progression"]

	assert.Contains(t, upstream, "player")
	assert.NotContains(t, upstream, "auth", "auth is player's detail; progression verifies tokens with the verifier cpbootstrap hands it")
}

// The bonuses module is downstream of planet, as missions is: it pays through planet's wallet, hears player's new
// players, and pays the boxes the boxes tell caught and the right answers the quizzes tell. Planet never names it.
func TestTheBonusesSpeakPlanetsPlayersTheBoxesAndTheQuizzesLanguageAlone(t *testing.T) {
	edges := moduleEdges(t)

	assert.ElementsMatch(t, []string{"planet", "player", "boxes", "quizzes"}, edges["bonuses"])
	assert.NotContains(t, edges["planet"], "bonuses", "planet knows nothing of the bonuses module")
}

// The boxes hear planet's clicks and read its room and its freeze. They tell a box caught and pay nothing: no module
// they speak to hears them back.
func TestTheBoxesSpeakPlanetsLanguageAlone(t *testing.T) {
	edges := moduleEdges(t)

	assert.ElementsMatch(t, []string{"planet"}, edges["boxes"])
	assert.NotContains(t, edges["planet"], "boxes", "planet knows nothing of the boxes")
}

// The quizzes hear planet's clicks, read its board and its freeze, and hear player's forgotten accounts. They tell an
// answer and pay nothing: no module they speak to hears them back.
func TestTheQuizzesSpeakPlanetsAndPlayersLanguageAlone(t *testing.T) {
	edges := moduleEdges(t)

	assert.ElementsMatch(t, []string{"planet", "player"}, edges["quizzes"])
	assert.NotContains(t, edges["planet"], "quizzes", "planet knows nothing of the quizzes")
	assert.NotContains(t, edges["player"], "quizzes", "player knows nothing of the quizzes")
}

func TestTheModulesThatPlayOrReadTheBonusesRulesSpeakItsLanguage(t *testing.T) {
	edges := moduleEdges(t)

	for _, module := range []string{"gamemaster", "promo"} {
		assert.Contains(t, edges[module], "bonuses", module)
	}
}

func TestTheModulesThatPlayHearOrReadTheBoxesSpeakTheirLanguage(t *testing.T) {
	edges := moduleEdges(t)

	for _, module := range []string{"gamemaster", "promo", "antibot", "missions", "progression", "bonuses"} {
		assert.Contains(t, edges[module], "boxes", module)
	}
}

func TestTheModulesThatPlayHearOrReadTheQuizzesSpeakTheirLanguage(t *testing.T) {
	edges := moduleEdges(t)

	for _, module := range []string{"gamemaster", "promo", "missions", "progression", "bonuses"} {
		assert.Contains(t, edges[module], "quizzes", module)
	}
}

// The flags module hears no module and calls none: planet keeps a catalog fed by its facts, and the others read the
// flags the game ships with and the shape of a code opened since (cpflags).
func TestTheFlagsSpeakNoModulesLanguageAndTheModulesThatCheckAFlagSpeakTheirs(t *testing.T) {
	edges := moduleEdges(t)

	assert.Empty(t, edges["flags"])
	for _, module := range []string{"planet", "chat", "scoring", "player", "alerts"} {
		assert.Contains(t, edges[module], "flags", module)
	}
}
