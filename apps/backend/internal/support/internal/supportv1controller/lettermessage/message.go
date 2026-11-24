package lettermessage

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

// Letter is a letter as the operator reads it.
func Letter(letter letters.Snapshot) *supportv1.Letter {
	message := &supportv1.Letter{
		Id: letter.ID.String(), SubmissionId: letter.Submission.String(), Address: string(letter.To),
		Subject: letter.Subject,
	}
	if letter.Account != (notes.AccountID{}) {
		message.AccountId = letter.Account.String()
	}
	for _, sent := range letter.Sent {
		message.Mails = append(message.Mails, &supportv1.LetterMail{Text: sent.Text, SentAt: timestamppb.New(sent.At)})
	}
	for _, reply := range letter.Replies {
		message.Replies = append(message.Replies, &supportv1.NoteReply{
			Text: reply.Text, From: string(reply.From), ReceivedAt: timestamppb.New(reply.At),
		})
	}
	if !letter.ClosedAt.IsZero() {
		message.ClosedAt = timestamppb.New(letter.ClosedAt)
	}
	return message
}
