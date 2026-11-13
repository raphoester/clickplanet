package parts

import (
	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// All is the one list of the quizzes' parts: a concept that reads the rules adds its Part here.
func All() cprules.Parts[*quizzesv1.Rules] {
	return cprules.Parts[*quizzesv1.Rules]{quizzes.Part}
}
