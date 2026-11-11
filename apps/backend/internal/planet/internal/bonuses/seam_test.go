package bonuses_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const planetPackages = "github.com/raphoester/clickplanet.lol-backend/internal/planet/"

// importsUnder is every import of each Go file under root, its tests included, that names one of planet's packages.
func importsUnder(t *testing.T, root string) map[string][]string {
	t.Helper()

	imports := map[string][]string{}
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err //nolint:wrapcheck // the test reads it.
		}
		for _, imported := range file.Imports {
			if name := strings.Trim(imported.Path.Value, `"`); strings.HasPrefix(name, planetPackages) {
				imports[path] = append(imports[path], strings.TrimPrefix(name, planetPackages))
			}
		}
		return nil
	}))
	return imports
}

// The offers leave planet for a module of their own (docs/offers-and-wallet.md): they reach planet through the ports
// they declare, never through one of its concepts, so that the move is a move.
func TestTheOffersImportNoConceptOfPlanets(t *testing.T) {
	for _, root := range []string{".", "../quizzes"} {
		for path, imported := range importsUnder(t, root) {
			for _, name := range imported {
				ours := strings.HasPrefix(name, "internal/bonuses") || name == "internal/quizzes"
				assert.True(t, ours, "%s imports %s", filepath.Join(root, path), name)
			}
		}
	}
}

func TestTheQuizzesAreTheOffersAlone(t *testing.T) {
	for path, imported := range importsUnder(t, "../..") {
		offers := strings.HasPrefix(path, "../../internal/bonuses/") || strings.HasPrefix(path, "../../internal/quizzes/") ||
			path == "../../module.go"
		for _, name := range imported {
			assert.False(t, name == "internal/quizzes" && !offers, "%s imports the quiz bank", path)
		}
	}
}
