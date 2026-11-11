package ratings

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// QuestionID names a question of the bank: its id there, which a regeneration keeps for the same question.
type QuestionID string

// AccountID is the account an answer came from. A caller with none, or an account forgotten, is NoAccount.
type AccountID = cpsession.AccountID

var NoAccount = cpsession.NoAccount

var ErrInvalidAccount = errors.New("not an account id")

func AccountIDOf(value string) (AccountID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return NoAccount, fmt.Errorf("%w: %q", ErrInvalidAccount, value)
	}
	return AccountID(id), nil
}

// Answer is one answer the ratings count: an opened question answered right or wrong, a timeout the page sent, or an
// answer past the deadline. A banner never opened, and a question opened and left to lapse, are not answers.
type Answer struct {
	Question QuestionID

	// Account is NoAccount for a caller with none: its answer still counts for the question.
	Account AccountID

	Correct bool

	// TimedOut is the page's own timeout: a choice past the three, which is wrong.
	TimedOut bool

	// After is how long after the question was opened the answer landed.
	After time.Duration

	At time.Time
}
