package main

import (
	"go/ast"
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

// The planet refuses a click and the chat a post or a reaction from one copy of the vendored VPN lists, ~2 MB parsed:
// describeModules builds it once and hands it to both, and neither module builds one of its own.
func TestThePlanetAndTheChatRefuseFromOneVPNList(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	require.NoError(t, err)

	built, handed := map[string]int{}, map[string]string{}
	for _, decl := range file.Decls {
		describe, ok := decl.(*ast.FuncDecl)
		if !ok || describe.Name.Name != "describeModules" {
			continue
		}
		ast.Inspect(describe.Body, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.AssignStmt:
				for i, value := range node.Rhs {
					if name, ok := node.Lhs[i].(*ast.Ident); ok && calls(value, "cpipblock", "New") {
						built[name.Name]++
					}
				}
			case *ast.CallExpr:
				for _, module := range []string{"planet", "chat"} {
					if !calls(node, module, "NewModule") || len(node.Args) != 2 {
						continue
					}
					if list, ok := node.Args[1].(*ast.Ident); ok {
						handed[module] = list.Name
					}
				}
			}
			return true
		})
	}

	require.Len(t, built, 1, "describeModules builds one VPN list")
	for list, times := range built {
		assert.Equal(t, 1, times)
		assert.Equal(t, map[string]string{"planet": list, "chat": list}, handed, "both modules are handed it")
	}

	for _, module := range []string{"planet", "chat"} {
		require.NoError(t, filepath.WalkDir(filepath.Join("../../internal", module), func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			source, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err //nolint:wrapcheck // the test reads it.
			}
			ast.Inspect(source, func(node ast.Node) bool {
				assert.False(t, calls(node, "cpipblock", "New"), "%s builds a VPN list of its own", path)
				return true
			})
			return nil
		}))
	}
}

func calls(node ast.Node, pkg string, function string) bool {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != function {
		return false
	}
	receiver, ok := selector.X.(*ast.Ident)
	return ok && receiver.Name == pkg
}
