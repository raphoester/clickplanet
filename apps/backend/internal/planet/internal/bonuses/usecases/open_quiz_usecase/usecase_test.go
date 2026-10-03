package open_quiz_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/open_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

var deadline = time.Date(2024, 1, 1, 0, 0, 5, 0, time.UTC)

func asPlayer(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest")
}

type fakeRegistry struct {
	asked bonuses.Asked
	known bool

	token      string
	forEntrant bonuses.Entrant
}

func (f *fakeRegistry) OpenQuiz(token string, entrant bonuses.Entrant) (bonuses.Asked, bool) {
	f.token, f.forEntrant = token, entrant

	return f.asked, f.known
}

func TestOpeningReadsTheQuestionAndTheClockItStarted(t *testing.T) {
	registry := &fakeRegistry{known: true, asked: bonuses.Asked{
		Question: "What is the capital of Estonia?",
		Options:  []string{"Riga", "Tallinn", "Vilnius"},
		Deadline: deadline,
		Window:   5 * time.Second,
	}}

	out, err := open_quiz_usecase.New(registry).Execute(asPlayer(t), open_quiz_usecase.In{Token: "t"})
	require.NoError(t, err)

	assert.Equal(t, "What is the capital of Estonia?", out.Question)
	assert.Equal(t, []string{"Riga", "Tallinn", "Vilnius"}, out.Options)
	assert.Equal(t, deadline, out.Deadline)
	assert.Equal(t, 5*time.Second, out.Window)

	assert.Equal(t, "t", registry.token)
	assert.Equal(t, bonuses.Entrant("1.2.3.4"), registry.forEntrant, "a guest's question is its scope's")
}

func TestAQuizThatIsNotThisCallersIsNotFound(t *testing.T) {
	_, err := open_quiz_usecase.New(&fakeRegistry{}).Execute(asPlayer(t), open_quiz_usecase.In{Token: "t"})

	require.ErrorIs(t, err, open_quiz_usecase.ErrNoSuchQuiz)
}
