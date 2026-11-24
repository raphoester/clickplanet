package write_letter_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters/usecases/write_letter_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var (
	now       = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	ada       = notes.AccountID{15: 1}
	bob       = notes.AccountID{15: 2}
	catalonia = letters.SubmissionID{15: 1}
	tyrol     = letters.SubmissionID{15: 2}
)

type stubPost struct {
	sent []letters.Mail
	err  error
}

func (s *stubPost) Send(_ context.Context, mail letters.Mail) (cpmail.MessageID, error) {
	if s.err != nil {
		return "", s.err
	}
	s.sent = append(s.sent, mail)
	return cpmail.MessageID([]string{"l1@clickplanet.lol", "l2@clickplanet.lol"}[len(s.sent)-1]), nil
}

func newUseCase(store *letters.MemoryStorage, post *stubPost) *write_letter_usecase.UseCase {
	return write_letter_usecase.New(&letters.SequentialIDs{}, notes.FixedAddressBook{ada: "ada@example.com"}, store, post,
		cptime.NewFixedClock(now))
}

func TestANewLetterGoesToTheAccountsAddressAndIsKept(t *testing.T) {
	store, post := letters.NewMemoryStorage(), &stubPost{}

	written, err := newUseCase(store, post).Execute(t.Context(), write_letter_usecase.In{
		Submission: catalonia.String(), Account: ada.String(), Subject: "Your flag Catalonia",
		Text: "Which city is its capital?",
	})
	require.NoError(t, err)

	assert.Equal(t, letters.Snapshot{
		ID: letters.LetterID{15: 1}, Submission: catalonia, Account: ada, To: "ada@example.com",
		Subject: "Your flag Catalonia", Sent: []letters.Sent{
			{Text: "Which city is its capital?", Mail: "l1@clickplanet.lol", At: now},
		},
	}, written)
	assert.Equal(t, []letters.Snapshot{written}, store.Kept())
	require.Len(t, post.sent, 1)
	assert.Equal(t, cpmail.Address("ada@example.com"), post.sent[0].To)
}

func TestAFurtherMailGoesInTheLettersThread(t *testing.T) {
	store, post := letters.NewMemoryStorage(), &stubPost{}
	useCase := newUseCase(store, post)
	first, err := useCase.Execute(t.Context(), write_letter_usecase.In{
		Submission: catalonia.String(), Account: ada.String(), Subject: "Your flag", Text: "Hi",
	})
	require.NoError(t, err)

	written, err := useCase.Execute(t.Context(), write_letter_usecase.In{Letter: first.ID.String(), Text: "Any news?"})
	require.NoError(t, err)

	assert.Len(t, written.Sent, 2)
	assert.Equal(t, "Re: Your flag", post.sent[1].Subject)
	assert.Equal(t, []cpmail.MessageID{"l1@clickplanet.lol"}, post.sent[1].Thread)
}

func TestASubmissionHasOneLetterAndAnotherSubmissionOfTheFounderItsOwn(t *testing.T) {
	store, post := letters.NewMemoryStorage(), &stubPost{}
	useCase := newUseCase(store, post)
	first, err := useCase.Execute(t.Context(), write_letter_usecase.In{
		Submission: catalonia.String(), Account: ada.String(), Subject: "Your flag Catalonia", Text: "Hi",
	})
	require.NoError(t, err)

	_, err = useCase.Execute(t.Context(), write_letter_usecase.In{
		Submission: catalonia.String(), Account: ada.String(), Subject: "Your flag Catalonia", Text: "Hi again",
	})
	require.ErrorIs(t, err, letters.ErrWrittenAlready)
	assert.Len(t, post.sent, 1, "nothing goes out")

	second, err := useCase.Execute(t.Context(), write_letter_usecase.In{
		Submission: tyrol.String(), Account: ada.String(), Subject: "Your flag Tyrol", Text: "Hi",
	})
	require.NoError(t, err)
	assert.Equal(t, []letters.Snapshot{first, second}, store.Kept())
}

func TestANewLetterNamesItsSubmission(t *testing.T) {
	for _, submission := range []string{"", "Catalonia", uuid.Nil.String()} {
		_, err := newUseCase(letters.NewMemoryStorage(), &stubPost{}).Execute(t.Context(), write_letter_usecase.In{
			Submission: submission, Account: ada.String(), Subject: "Your flag", Text: "Hi",
		})
		require.ErrorIs(t, err, letters.ErrInvalidSubmission, "%q", submission)
	}
}

func TestAnAccountWithNoAddressGetsNoLetterAndAMailThatDidNotGoKeepsNothing(t *testing.T) {
	store := letters.NewMemoryStorage()

	_, err := newUseCase(store, &stubPost{}).Execute(t.Context(), write_letter_usecase.In{
		Submission: catalonia.String(), Account: bob.String(), Subject: "Your flag", Text: "Hi",
	})
	require.ErrorIs(t, err, notes.ErrNoReplyAddress)

	_, err = newUseCase(store, &stubPost{err: notes.ErrNotSent}).Execute(t.Context(), write_letter_usecase.In{
		Submission: catalonia.String(), Account: ada.String(), Subject: "Your flag", Text: "Hi",
	})
	require.ErrorIs(t, err, notes.ErrNotSent)
	assert.Empty(t, store.Kept())
}
