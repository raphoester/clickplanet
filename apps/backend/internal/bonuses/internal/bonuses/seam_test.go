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

const quizBank = "github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/quizzes"

// The operator's list of the questions (bonuses.v1.AdminService/ListQuizQuestions), on the admin listener alone, shows
// each right answer on purpose.
const operatorsList = "../../internal/bonusesv1controller/list_quiz_questions_handler/quiz_questions_query/inmemory_quiz_questions/"

// The quiz bank holds the answers: only the offers, the module that loads it and the operator's list may read it.
func TestTheQuizzesAreTheOffersAlone(t *testing.T) {
	require.NoError(t, filepath.WalkDir("../..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err //nolint:wrapcheck // the test reads it.
		}
		offers := strings.HasPrefix(path, "../../internal/bonuses/") || strings.HasPrefix(path, "../../internal/quizzes/") ||
			path == "../../module.go" || strings.HasPrefix(path, operatorsList)
		for _, imported := range file.Imports {
			assert.False(t, strings.Trim(imported.Path.Value, `"`) == quizBank && !offers, "%s imports the quiz bank", path)
		}
		return nil
	}))
}
