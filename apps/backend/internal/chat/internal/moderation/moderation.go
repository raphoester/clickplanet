// Package moderation is the chat's moderation history: what an operator did to an account (a mute, an unmute, a
// removal), with the reason given and who did it, kept past the chat's retention.
package moderation

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

// What an action did.
type Kind uint8

const (
	Muted Kind = iota + 1
	Unmuted
	Removed
)

// As postgres keeps it: a value of the type moderation_action.
func (k Kind) String() string {
	switch k {
	case Muted:
		return "mute"
	case Unmuted:
		return "unmute"
	case Removed:
		return "removal"
	default:
		return ""
	}
}

// Who did it: the operator's account, and the app it connected ("Claude"), empty for its own page. Nobody for a call
// on the console listener.
type Operator struct {
	account messages.AccountID
	client  string
}

func OperatorOf(account messages.AccountID, client string) Operator {
	return Operator{account: account, client: client}
}

func (o Operator) Account() messages.AccountID { return o.account }

func (o Operator) Client() string { return o.client }

// One action on one account: the account it was taken on, never the network a mute also holds.
type Action struct {
	snapshot Snapshot
}

type Snapshot struct {
	Account messages.AccountID
	At      time.Time
	Kind    Kind
	// A mute's and an unmute's.
	MuteKind mutes.Kind
	// A mute's.
	Duration time.Duration
	// A removal's: the message, and a copy of what it said, without its address.
	Message messages.MessageID
	Text    string
	Gif     messages.Gif
	Reason  Reason
	By      Operator
}

func NewMute(mute mutes.Mute, reason Reason, by Operator) Action {
	return Action{snapshot: Snapshot{
		Account:  mute.Caller().Account(),
		At:       mute.At(),
		Kind:     Muted,
		MuteKind: mute.Kind(),
		Duration: mute.Duration(),
		Reason:   reason,
		By:       by,
	}}
}

func NewUnmute(account messages.AccountID, kind mutes.Kind, at time.Time, reason Reason, by Operator) Action {
	return Action{snapshot: Snapshot{
		Account:  account,
		At:       at,
		Kind:     Unmuted,
		MuteKind: kind,
		Reason:   reason,
		By:       by,
	}}
}

func NewRemoval(message messages.Message, at time.Time, reason Reason, by Operator) Action {
	removed := message.TakeSnapshot()
	return Action{snapshot: Snapshot{
		Account: removed.Account,
		At:      at,
		Kind:    Removed,
		Message: removed.ID,
		Text:    removed.Text,
		Gif:     removed.Gif,
		Reason:  reason,
		By:      by,
	}}
}

func (a Action) TakeSnapshot() Snapshot { return a.snapshot }

// The store only writes: what it keeps is read by the operator's query (moderation_query).
type Storage interface {
	// A removal of a message already removed keeps the first: a call tried again never counts twice.
	Record(ctx context.Context, action Action) error
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
	// Deletes the actions taken on the account, and takes it off the actions it took as an operator.
	Forget(ctx context.Context, account messages.AccountID) error
}
