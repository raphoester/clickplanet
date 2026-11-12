package inmemory_quiz_questions_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/list_quiz_questions_handler/quiz_questions_query/inmemory_quiz_questions"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
)

const fixture = `{"format": 1, "questions": [
  {"id":"capital:ee","subject":"ee","text":"What is the capital of Estonia?","answer":"Tallinn",
   "wrong":["Riga","Vilnius"],"namesSubject":true,"prior":0.45},
  {"id":"capital:lv","subject":"lv","text":"What is the capital of Latvia?","answer":"Riga",
   "wrong":["Tallinn","Vilnius"],"namesSubject":true}
]}`

func TestEachQuestionIsListedWithItsRatingAndWhatItsAnswersMeasured(t *testing.T) {
	moved := ratings.PriorOf(0.45).Answered(true).Answered(false).Answered(false).Answered(true)
	bank, err := quizzes.LoadFixture([]byte(fixture), nil,
		ratings.NewBook(map[ratings.QuestionID]ratings.Rating{"capital:ee": moved}))
	require.NoError(t, err)

	listed := inmemory_quiz_questions.New(bank).Questions()

	require.Len(t, listed, 2)
	assert.True(t, proto.Equal(&bonusesv1.QuizQuestion{
		Id: "capital:ee", Text: "What is the capital of Estonia?", Answer: "Tallinn", Subject: "ee",
		Prior: proto.Float64(0.45), Rating: moved.Level(), PredictedShare: moved.Chance(),
		Answers: 4, Right: 2, MeasuredShare: proto.Float64(0.5),
	}, listed[0]), "%v", listed[0])

	fallback := ratings.PriorOf(quizzes.FallbackPrior)
	assert.True(t, proto.Equal(&bonusesv1.QuizQuestion{
		Id: "capital:lv", Text: "What is the capital of Latvia?", Answer: "Riga", Subject: "lv",
		Rating: fallback.Level(), PredictedShare: fallback.Chance(),
	}, listed[1]), "no prior in the bank, no answer yet: %v", listed[1])
}
