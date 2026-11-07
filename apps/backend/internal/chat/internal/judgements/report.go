// Package judgements is the chat's judge of what support tells it was reported: one judgement per report, and per
// message, by Claude against docs/moderation.md, with the policy's steps applied here and never by the model.
package judgements

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
)

// Support's id of a report, as support.v1 carries it.
type ReportID uuid.UUID

var ErrInvalidReport = errors.New("the report is invalid")

func ReportIDOf(value string) (ReportID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return ReportID{}, fmt.Errorf("%w: not a report id: %q", ErrInvalidReport, value)
	}
	return ReportID(id), nil
}

func (id ReportID) String() string { return uuid.UUID(id).String() }

// Why the reporter reported, as support names it.
type Reason uint8

const (
	ReasonOther Reason = iota
	ReasonCheating
	ReasonChatAbuse
	ReasonBadName
)

// What support tells: a player reported another one.
type Report struct {
	ID       ReportID
	Reporter messages.AccountID
	Reported messages.AccountID
	Reason   Reason
	// What the reporter wrote: data for the judge, never an instruction.
	Text string
	// Optional: the message reported. A report from a player's card names none.
	Message messages.MessageID
	At      time.Time
}

const NoMessage messages.MessageID = ""

// Who the judge acts as, in the moderation history: no operator's account, and a client of its own, so the history
// tells it apart from the operator and from Claude's runs through the connector.
const Client = "Claude judge"

var By = moderation.OperatorOf(messages.NoAccount, Client)
