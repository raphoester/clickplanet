package uuid_submission_ids

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

type Provider struct{}

var _ submissions.IDProvider = Provider{}

func (Provider) NewID() (submissions.SubmissionID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return submissions.SubmissionID{}, fmt.Errorf("failed to draw a submission id: %w", err)
	}
	return submissions.SubmissionID(id), nil
}
