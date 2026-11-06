// Package renames keeps every username an operator gave an account, with the old one, the reason given and who did
// it: the moderation history of the names, read by the operator (ListRenames).
package renames

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

// Who did it: the operator's account, and the app it connected ("Claude"), empty for its own page. Nobody for a call
// on the console listener.
type Operator struct {
	account players.AccountID
	client  string
}

func OperatorOf(account players.AccountID, client string) Operator {
	return Operator{account: account, client: client}
}

func (o Operator) Account() players.AccountID { return o.account }

func (o Operator) Client() string { return o.client }

type Rename struct {
	snapshot Snapshot
}

type Snapshot struct {
	Account players.AccountID
	At      time.Time
	Old     players.Name
	New     players.Name
	Reason  Reason
	By      Operator
}

// From the profile before the rename to the profile after it.
func NewRename(before players.Profile, after players.Profile, at time.Time, reason Reason, by Operator) Rename {
	old, renamed := before.TakeSnapshot(), after.TakeSnapshot()
	return Rename{snapshot: Snapshot{
		Account: renamed.Account, At: at, Old: old.Name, New: renamed.Name, Reason: reason, By: by,
	}}
}

func (r Rename) TakeSnapshot() Snapshot { return r.snapshot }

// The store only writes: what it keeps is read by the operator's query (renames_query).
type Storage interface {
	Record(ctx context.Context, rename Rename) error
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
	// Deletes the account's renames, and takes it off the renames it made as an operator.
	Forget(ctx context.Context, account players.AccountID) error
}
