package submissionmessage

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

// A reviewer opens a submission within the hour its picture's link is good for.
const heldLinkTTL = time.Hour

type Links interface {
	HeldURL(key string, ttl time.Duration) (string, error)
}

func New(flags flagmessage.Messages, links Links) Messages {
	return Messages{flags: flags, links: links}
}

// Messages writes a submission as the wire carries it: to its player, and to the operator with its picture held.
type Messages struct {
	flags flagmessage.Messages
	links Links
}

func (m Messages) Submission(submission submissions.Snapshot) *flagsv1.Submission {
	message := &flagsv1.Submission{
		Id: submission.ID.String(), Name: submission.Name.String(), Pin: flagmessage.Pin(submission.Pin),
		Flag: submission.Flag.String(), State: State(submission.State),
		SubmittedAt: timestamppb.New(submission.SubmittedAt), DecidedAt: timestamppb.New(submission.DecidedAt),
		Rejection: rejection(submission), Paid: submission.Paid,
	}
	if public(submission.State) {
		message.Picture = m.flags.Picture(submission.Picture)
	}
	return message
}

func (m Messages) Kept(submission submissions.Snapshot) (*flagsv1.KeptSubmission, error) {
	message := &flagsv1.KeptSubmission{
		Id: submission.ID.String(), Name: submission.Name.String(), Pin: flagmessage.Pin(submission.Pin),
		Flag: submission.Flag.String(), State: State(submission.State),
		SubmittedAt: timestamppb.New(submission.SubmittedAt), DecidedAt: timestamppb.New(submission.DecidedAt),
		Rejection: rejection(submission), OtherColorsPercent: uint32(submission.OtherColors),
		AccountId: submission.Account.String(),
	}
	switch {
	case submission.Held:
		url, err := m.links.HeldURL(submission.ID.PictureKey(), heldLinkTTL)
		if err != nil {
			return nil, err //nolint:wrapcheck // the bucket names it.
		}
		message.Picture = flagmessage.PictureAt(url, submission.Picture)
	case public(submission.State):
		message.Picture = m.flags.Picture(submission.Picture)
	}
	return message, nil
}

func public(state submissions.State) bool {
	return state == submissions.Approved || state == submissions.Opened
}

func rejection(submission submissions.Snapshot) *flagsv1.Rejection {
	if submission.State != submissions.Rejected {
		return nil
	}
	return &flagsv1.Rejection{Reason: Reason(submission.Rejection.Reason), Note: submission.Rejection.Note}
}

func State(state submissions.State) flagsv1.SubmissionState {
	switch state {
	case submissions.Waiting:
		return flagsv1.SubmissionState_SUBMISSION_STATE_WAITING
	case submissions.Approved:
		return flagsv1.SubmissionState_SUBMISSION_STATE_APPROVED
	case submissions.Rejected:
		return flagsv1.SubmissionState_SUBMISSION_STATE_REJECTED
	case submissions.Opened:
		return flagsv1.SubmissionState_SUBMISSION_STATE_OPENED
	}
	return flagsv1.SubmissionState_SUBMISSION_STATE_UNSPECIFIED
}

func Reason(reason submissions.Reason) flagsv1.RejectionReason {
	switch reason {
	case submissions.NotAPlace:
		return flagsv1.RejectionReason_REJECTION_REASON_NOT_A_PLACE
	case submissions.Exists:
		return flagsv1.RejectionReason_REJECTION_REASON_EXISTS
	case submissions.Unreadable:
		return flagsv1.RejectionReason_REJECTION_REASON_UNREADABLE
	case submissions.Hateful:
		return flagsv1.RejectionReason_REJECTION_REASON_HATEFUL
	case submissions.Other:
		return flagsv1.RejectionReason_REJECTION_REASON_OTHER
	}
	return flagsv1.RejectionReason_REJECTION_REASON_UNSPECIFIED
}

// ReasonOf is the reason the wire names; none for one it does not, which the rejection refuses.
func ReasonOf(reason flagsv1.RejectionReason) submissions.Reason {
	switch reason {
	case flagsv1.RejectionReason_REJECTION_REASON_NOT_A_PLACE:
		return submissions.NotAPlace
	case flagsv1.RejectionReason_REJECTION_REASON_EXISTS:
		return submissions.Exists
	case flagsv1.RejectionReason_REJECTION_REASON_UNREADABLE:
		return submissions.Unreadable
	case flagsv1.RejectionReason_REJECTION_REASON_HATEFUL:
		return submissions.Hateful
	case flagsv1.RejectionReason_REJECTION_REASON_OTHER:
		return submissions.Other
	case flagsv1.RejectionReason_REJECTION_REASON_UNSPECIFIED:
		return ""
	}
	return ""
}
