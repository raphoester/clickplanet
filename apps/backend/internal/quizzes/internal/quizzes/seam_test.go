package quizzes_test

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

const quizBank = "github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/bank"

// The operator's list of the questions (quizzes.v1.AdminService/ListQuizQuestions), on the admin listener alone, shows
// each right answer on purpose.
const operatorsList = "../../internal/quizzesv1controller/list_quiz_questions_handler/quiz_questions_query/inmemory_quiz_questions/"

func TestTheBankIsTheRegistrysAlone(t *testing.T) {
	require.NoError(t, filepath.WalkDir("../..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err //nolint:wrapcheck // the test reads it.
		}
		allowed := strings.HasPrefix(path, "../../internal/quizzes/") || strings.HasPrefix(path, "../../internal/bank/") ||
			path == "../../module.go" || strings.HasPrefix(path, operatorsList)
		for _, imported := range file.Imports {
			assert.False(t, strings.Trim(imported.Path.Value, `"`) == quizBank && !allowed, "%s imports the quiz bank", path)
		}
		return nil
	}))
}
