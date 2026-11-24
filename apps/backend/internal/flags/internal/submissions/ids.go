package submissions

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrUnknownSubmission = errors.New("no submission was kept under this id")
	ErrInvalidAccount    = errors.New("not the id of an account")
)

type SubmissionID uuid.UUID

func ParseSubmissionID(s string) (SubmissionID, error) {
	id, err := uuid.Parse(s)
	if err != nil || id == uuid.Nil {
		return SubmissionID{}, fmt.Errorf("%w: %q", ErrUnknownSubmission, s)
	}
	return SubmissionID(id), nil
}

func (id SubmissionID) String() string {
	return uuid.UUID(id).String()
}

// PictureKey is where the bucket keeps the submission's picture, private, until its approval.
func (id SubmissionID) PictureKey() string {
	return "flags/submissions/" + id.String() + ".png"
}

type IDProvider interface {
	NewID() (SubmissionID, error)
}

// AccountID is the account that submitted a flag: its founder.
type AccountID uuid.UUID

func ParseAccountID(s string) (AccountID, error) {
	id, err := uuid.Parse(s)
	if err != nil || id == uuid.Nil {
		return AccountID{}, fmt.Errorf("%w: %q", ErrInvalidAccount, s)
	}
	return AccountID(id), nil
}

func (id AccountID) String() string {
	return uuid.UUID(id).String()
}
