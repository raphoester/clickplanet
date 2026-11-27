package review_asked_subscriber

import (
	"context"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/reviews"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, review reviews.Review) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*flagsv1.ReviewAsked] = Subscriber{}

// Handle warns the staff once of a flag that waits on a person. A cause this build does not know is left.
func (s Subscriber) Handle(ctx context.Context, event *flagsv1.ReviewAsked) error {
	cause, ok := causeOf(event.GetCause())
	if !ok {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, reviews.Review{ //nolint:wrapcheck // the use case named it.
		Submission: reviews.SubmissionID(event.GetSubmissionId()), Name: event.GetName(), Cause: cause,
		Why: event.GetWhy(), AskedAt: event.GetAskedAt().AsTime(),
	})
}

func causeOf(cause flagsv1.ReviewCause) (reviews.Cause, bool) {
	switch cause {
	case flagsv1.ReviewCause_REVIEW_CAUSE_APPEAL:
		return reviews.Appeal, true
	case flagsv1.ReviewCause_REVIEW_CAUSE_UNSURE:
		return reviews.Unsure, true
	case flagsv1.ReviewCause_REVIEW_CAUSE_UNSPECIFIED:
	}
	return "", false
}
