package grants

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

var (
	ErrInvalidGrant   = errors.New("not the id of a payment")
	ErrInvalidPeriod  = errors.New("not a period of time sold")
	ErrInvalidTarget  = errors.New("time goes to a submission or a flag")
	ErrNotTheFlag     = errors.New("the flag is not the one the submission opened")
	ErrInvalidGranted = errors.New("a grant kept in no state it can be in")
	ErrNotHeld        = errors.New("the payment's time is not held for a review")
	ErrStillWaiting   = errors.New("held time is released once its submission is decided")
)

const maxGrantID = 255

// GrantID names the payment a grant is for: told twice, a payment grants once.
type GrantID string

func ParseGrantID(s string) (GrantID, error) {
	if s == "" || len(s) > maxGrantID || strings.IndexFunc(s, unicode.IsSpace) >= 0 {
		return "", fmt.Errorf("%w: %q", ErrInvalidGrant, s)
	}
	return GrantID(s), nil
}

func (id GrantID) String() string {
	return string(id)
}

// Period is how much time a payment buys: a week, or a calendar month or year, as a plan renews.
type Period string

const (
	Week  Period = "week"
	Month Period = "month"
	Year  Period = "year"
)

func (p Period) After(t time.Time) time.Time {
	switch p {
	case Week:
		return t.AddDate(0, 0, 7)
	case Month:
		return t.AddDate(0, 1, 0)
	case Year:
		return t.AddDate(1, 0, 0)
	}
	return t
}

func (p Period) check() error {
	switch p {
	case Week, Month, Year:
		return nil
	}
	return fmt.Errorf("%w: %q", ErrInvalidPeriod, p)
}

// Target is what paid time goes to: a submission of a new flag, or a flag opened since.
type Target struct {
	submission submissions.SubmissionID
	flag       catalog.FlagID
}

func ParseTarget(submission, flag string) (Target, error) {
	switch {
	case submission != "" && flag == "":
		id, err := submissions.ParseSubmissionID(submission)
		if err != nil {
			return Target{}, err //nolint:wrapcheck // the parse names it.
		}
		return Target{submission: id}, nil
	case flag != "" && submission == "":
		id, err := catalog.ParseFlagID(flag)
		if err != nil {
			return Target{}, err //nolint:wrapcheck // the parse names it.
		}
		return Target{flag: id}, nil
	}
	return Target{}, fmt.Errorf("%w: submission %q, flag %q", ErrInvalidTarget, submission, flag)
}

// Submission is the submission the time is for, when it is for one.
func (t Target) Submission() (submissions.SubmissionID, bool) {
	return t.submission, t.flag == ""
}

// Flag is the flag the time is for, when it is for one rather than a submission.
func (t Target) Flag() (catalog.FlagID, bool) {
	return t.flag, t.flag != ""
}

// Paid is a payment as the shop tells it: what it is for, how much time it bought, and when.
type Paid struct {
	Grant  GrantID
	Target Target
	Period Period
	At     time.Time
}

func ParsePaid(grant, submission, flag string, period Period, at time.Time) (Paid, error) {
	id, err := ParseGrantID(grant)
	if err != nil {
		return Paid{}, err
	}
	target, err := ParseTarget(submission, flag)
	if err != nil {
		return Paid{}, err
	}
	if err := period.check(); err != nil {
		return Paid{}, err
	}
	return Paid{Grant: id, Target: target, Period: period, At: at}, nil
}

// Grant is a payment's time, on the flag it went to, from its payment to a refund that takes it back. A payment taken
// back before it was heard grants nothing. One for a submission that waits is held until its approval.
type Grant struct {
	id GrantID
	// Held.
	held       bool
	submission submissions.SubmissionID
	period     Period
	heldAt     time.Time
	// Granted.
	granted   bool
	flag      catalog.FlagID
	added     time.Duration
	grantedAt time.Time
	// Taken back.
	takenBack   bool
	takenBackAt time.Time
}

// Unheard is a payment no grant was kept for yet.
func Unheard(id GrantID) Grant {
	return Grant{id: id}
}

// Heard is true once the payment's time was held, granted, or taken back: the next word of it changes nothing.
func (g Grant) Heard() bool {
	return g.held || g.granted || g.takenBack
}

// TakenBackAlready is true once the payment was taken back: told again, it takes back nothing more.
func (g Grant) TakenBackAlready() bool {
	return g.takenBack
}

// ToSubmission grants the time of a payment for a submission: one that waits holds it until its approval, an approved
// one opens its flag for the period from now, an opened one adds the period to its flag, read beside it (the zero flag
// otherwise).
func (g Grant) ToSubmission(
	submission submissions.Submission, flag catalog.Flag, opening catalog.Opening, names catalog.Names, period Period,
	now time.Time,
) (Grant, submissions.Submission, catalog.Flag, []proto.Message, error) {
	if submission.Waiting() {
		g.held, g.submission, g.period, g.heldAt = true, submission.TakeSnapshot().ID, period, now
		return g, submission, flag, nil, nil
	}
	return g.toDecided(submission, flag, opening, names, period, now)
}

// Released grants the time held for a submission once it is approved, from now: its flag opens for the period, or ends
// that much later when another payment opened it first.
func (g Grant) Released(
	submission submissions.Submission, flag catalog.Flag, opening catalog.Opening, names catalog.Names, now time.Time,
) (Grant, submissions.Submission, catalog.Flag, []proto.Message, error) {
	if !g.held || g.granted || g.takenBack {
		return g, submission, flag, nil, fmt.Errorf("%w: %s", ErrNotHeld, g.id)
	}
	if submission.Waiting() {
		return g, submission, flag, nil, fmt.Errorf("%w: %s", ErrStillWaiting, g.submission)
	}
	return g.toDecided(submission, flag, opening, names, g.period, now)
}

func (g Grant) toDecided(
	submission submissions.Submission, flag catalog.Flag, opening catalog.Opening, names catalog.Names, period Period,
	now time.Time,
) (Grant, submissions.Submission, catalog.Flag, []proto.Message, error) {
	opened, ok := submission.OpenedFlag()
	if ok {
		if flag.TakeSnapshot().ID != opened {
			return g, submission, flag, nil, fmt.Errorf("%w: %s", ErrNotTheFlag, opened)
		}
		granted, extended, facts, err := g.ToFlag(flag, period, now)
		return granted, submission, extended, facts, err
	}
	submission, flag, err := submission.Open(opening, names, period.After(now))
	if err != nil {
		return g, submission, flag, nil, err //nolint:wrapcheck // the submission names it.
	}
	g.granted, g.flag, g.added, g.grantedAt = true, flag.TakeSnapshot().ID, period.After(now).Sub(now), now
	return g, submission, flag, flag.Facts(), nil
}

// ToFlag grants the time of a payment to a flag: an open one ends a period later, one closed less than 30 days ago
// opens again for the period, and is told opened.
func (g Grant) ToFlag(flag catalog.Flag, period Period, now time.Time) (Grant, catalog.Flag, []proto.Message, error) {
	extended, added, err := flag.Extended(period, now)
	if err != nil {
		return g, flag, nil, err //nolint:wrapcheck // the flag names it.
	}
	g.granted, g.flag, g.added, g.grantedAt = true, flag.TakeSnapshot().ID, added, now
	if flag.TakeSnapshot().State == extended.TakeSnapshot().State {
		return g, extended, nil, nil
	}
	return g, extended, extended.Facts(), nil
}

// TakenBack is the grant once its payment was refunded or lost in a dispute. Once granted, the flag it went to is read
// beside it and ends that much sooner, told closed when that end is past; before, held or not, the grant comes to
// nothing.
func (g Grant) TakenBack(flag catalog.Flag, now time.Time) (Grant, catalog.Flag, []proto.Message, error) {
	g.takenBack, g.takenBackAt = true, now
	if !g.granted {
		return g, flag, nil, nil
	}
	if flag.TakeSnapshot().ID != g.flag {
		return g, flag, nil, fmt.Errorf("%w: %s", ErrNotTheFlag, g.flag)
	}
	shortened := flag.Shortened(g.added, now)
	if flag.TakeSnapshot().State == shortened.TakeSnapshot().State {
		return g, shortened, nil, nil
	}
	return g, shortened, shortened.Facts(), nil
}

// GrantedFlag is the flag the payment's time went to, once granted.
func (g Grant) GrantedFlag() (catalog.FlagID, bool) {
	return g.flag, g.granted
}

func (g Grant) TakeSnapshot() Snapshot {
	return Snapshot{
		ID: g.id, Held: g.held, Submission: g.submission, Period: g.period, HeldAt: g.heldAt, Granted: g.granted,
		Flag: g.flag, Added: g.added, GrantedAt: g.grantedAt, TakenBack: g.takenBack, TakenBackAt: g.takenBackAt,
	}
}

type Snapshot struct {
	ID GrantID
	// The payment's Period was held for Submission while it waited for its review; released once Granted too.
	Held       bool
	Submission submissions.SubmissionID
	Period     Period
	HeldAt     time.Time
	// The payment's time went to Flag, and Added was its length there.
	Granted   bool
	Flag      catalog.FlagID
	Added     time.Duration
	GrantedAt time.Time
	// The payment was refunded or lost in a dispute.
	TakenBack   bool
	TakenBackAt time.Time
}

func (s Snapshot) Restore() (Grant, error) {
	if _, err := ParseGrantID(s.ID.String()); err != nil {
		return Grant{}, err
	}
	if s.Granted && (s.Flag == "" || s.Added <= 0) {
		return Grant{}, fmt.Errorf("%w: %s granted no time to no flag", ErrInvalidGranted, s.ID)
	}
	if !s.Granted && (s.Flag != "" || s.Added != 0) {
		return Grant{}, fmt.Errorf("%w: %s not granted, with a flag or a length", ErrInvalidGranted, s.ID)
	}
	if s.Held {
		if s.Submission == (submissions.SubmissionID{}) {
			return Grant{}, fmt.Errorf("%w: %s held for no submission", ErrInvalidGranted, s.ID)
		}
		if err := s.Period.check(); err != nil {
			return Grant{}, fmt.Errorf("%w: %s held: %w", ErrInvalidGranted, s.ID, err)
		}
	}
	if !s.Held && (s.Submission != (submissions.SubmissionID{}) || s.Period != "") {
		return Grant{}, fmt.Errorf("%w: %s not held, with a submission or a period", ErrInvalidGranted, s.ID)
	}
	return Grant{
		id: s.ID, held: s.Held, submission: s.Submission, period: s.Period, heldAt: s.HeldAt, granted: s.Granted,
		flag: s.Flag, added: s.Added, grantedAt: s.GrantedAt, takenBack: s.TakenBack, takenBackAt: s.TakenBackAt,
	}, nil
}
