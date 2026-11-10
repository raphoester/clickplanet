// Whether a player's name may be shown in the game's posts. TikTok and YouTube take down a post that names a terrorist
// group, a hate code or a mass killer, and strike the channel; the game itself shows every name. Each name is judged
// once, and judged again when its player renames.
package verdicts

import (
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type AccountID = cpsession.AccountID

// Who decided a verdict.
type Decider string

const (
	ByJudge    Decider = "judge"
	ByOperator Decider = "operator"
)

var ErrUnknownDecider = errors.New("not who decides a verdict")

func DeciderOf(value string) (Decider, error) {
	switch Decider(value) {
	case ByJudge, ByOperator:
		return Decider(value), nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownDecider, value)
}

type Verdict struct {
	account AccountID
	name    string
	shown   bool
	reason  string
	by      Decider
	at      time.Time
}

type Snapshot struct {
	Account AccountID
	Name    string
	Shown   bool
	// Why a name is hidden; empty for a name shown.
	Reason string
	By     Decider
	At     time.Time
}

func (v Verdict) TakeSnapshot() Snapshot {
	return Snapshot{Account: v.account, Name: v.name, Shown: v.shown, Reason: v.reason, By: v.by, At: v.at}
}

func (s Snapshot) Restore() Verdict {
	return Verdict{account: s.Account, name: s.Name, shown: s.Shown, reason: s.Reason, by: s.By, at: s.At}
}

// What the judge answered of one name.
type Judgement struct {
	Name   string
	Shown  bool
	Reason string
}

var (
	ErrNoJudge   = errors.New("no judge of the names: every name is hidden")
	ErrNoVerdict = errors.New("no verdict of this player's name")
)

// The judgements of names, as the verdicts of the accounts named so: a name the judge did not answer has none.
func Judged(names map[AccountID]string, judgements []Judgement, at time.Time) []Verdict {
	byName := make(map[string]Judgement, len(judgements))
	for _, judgement := range judgements {
		byName[judgement.Name] = judgement
	}
	var judged []Verdict
	for account, name := range names {
		judgement, answered := byName[name]
		if !answered {
			continue
		}
		reason := judgement.Reason
		if judgement.Shown {
			reason = ""
		}
		judged = append(judged, Verdict{
			account: account, name: name, shown: judgement.Shown, reason: reason, by: ByJudge, at: at.UTC(),
		})
	}
	return judged
}

// The operator's verdict of the same name: it holds until the player renames.
func (v Verdict) Overruled(shown bool, reason string, at time.Time) Verdict {
	if shown {
		reason = ""
	}
	return Verdict{account: v.account, name: v.name, shown: shown, reason: reason, by: ByOperator, at: at.UTC()}
}

// The names the posts may show, and those no verdict covers yet: a verdict of a name the player left covers nothing.
func Screen(names map[AccountID]string, kept []Verdict) (shown, unjudged map[AccountID]string) {
	byAccount := make(map[AccountID]Verdict, len(kept))
	for _, verdict := range kept {
		byAccount[verdict.account] = verdict
	}
	shown, unjudged = map[AccountID]string{}, map[AccountID]string{}
	for account, name := range names {
		verdict, judged := byAccount[account]
		switch {
		case !judged || verdict.name != name:
			unjudged[account] = name
		case verdict.shown:
			shown[account] = name
		}
	}
	return shown, unjudged
}
