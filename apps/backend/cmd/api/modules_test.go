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

// A module depends on another when its code speaks the other's proto package: it calls it, hears its events or sends
// it commands. The deprecated mint's package is auth's.
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
				match := protoPackage.FindStringSubmatch(strings.Trim(imported.Path.Value, `"`))
				if match == nil {
					continue
				}
				owner := match[1]
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

// Auth is player's detail: the game asks player who an account is. Alerts is the one exception, for the Discord join's
// token, which auth keeps sealed and never puts on Redis.
func TestOnlyPlayerAndAlertsSpeakAuthsLanguage(t *testing.T) {
	var speakers []string
	for module, upstream := range moduleEdges(t) {
		if slices.Contains(upstream, "auth") {
			speakers = append(speakers, module)
		}
	}
	slices.Sort(speakers)

	assert.Equal(t, []string{"alerts", "player"}, speakers)
}

func TestProgressionKnowsPlayerAndNotAuth(t *testing.T) {
	upstream := moduleEdges(t)["progression"]

	assert.Contains(t, upstream, "player")
	assert.NotContains(t, upstream, "auth", "auth is player's detail; progression verifies tokens with cpsessionverifier alone")
}
