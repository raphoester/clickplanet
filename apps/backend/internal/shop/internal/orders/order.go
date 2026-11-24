package orders

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
)

var (
	ErrGuest          = errors.New("only a signed-in account pays: a guest's is deleted after 90 days away")
	ErrInvalidAccount = errors.New("not the id of an account")
	ErrInvalidTarget  = errors.New("time is bought for one submission or one flag")
	ErrNoAgreement    = errors.New("the buyer must ask for the flag to start at once, and so end its right to cancel")
	ErrNotOurs        = errors.New("not an order of the shop's")
)

// AccountID is the account that pays: the founder of the flag.
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

// BuyerOf is the account a call came from, when it may pay: a flag's time needs a founder that lasts.
func BuyerOf(account string, linked bool) (AccountID, error) {
	id, err := ParseAccountID(account)
	if err != nil {
		return AccountID{}, fmt.Errorf("%w: %w", ErrGuest, err)
	}
	if !linked {
		return AccountID{}, fmt.Errorf("%w: %s", ErrGuest, id)
	}
	return id, nil
}

// Target is what the time is bought for: the approved submission of a new flag, or a flag opened since. The shop keeps
// its id as the flags module gave it; the flags module checks it.
type Target struct {
	submission string
	flag       string
}

var (
	submissionShape = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	flagShape       = regexp.MustCompile(`^r-[a-z0-9]{10}$|^[a-z]{2}$|^gb-[a-z]{3}$`)
)

func ParseTarget(submission, flag string) (Target, error) {
	switch {
	case submission != "" && flag == "" && submissionShape.MatchString(submission):
		return Target{submission: submission}, nil
	case flag != "" && submission == "" && flagShape.MatchString(flag):
		return Target{flag: flag}, nil
	}
	return Target{}, fmt.Errorf("%w: submission %q, flag %q", ErrInvalidTarget, submission, flag)
}

// Submission is the submission's id, or empty for a flag.
func (t Target) Submission() string {
	return t.submission
}

// Flag is the flag's code, or empty for a submission.
func (t Target) Flag() string {
	return t.flag
}

// TargetKind names a target's kind where it is kept.
type TargetKind string

const (
	SubmissionTarget TargetKind = "submission"
	FlagTarget       TargetKind = "flag"
)

// Kept is the target's kind and id, as a store keeps them.
func (t Target) Kept() (TargetKind, string) {
	if t.submission != "" {
		return SubmissionTarget, t.submission
	}
	return FlagTarget, t.flag
}

func TargetOf(kind TargetKind, id string) (Target, error) {
	switch kind {
	case SubmissionTarget:
		return ParseTarget(id, "")
	case FlagTarget:
		return ParseTarget("", id)
	}
	return Target{}, fmt.Errorf("%w: a %q", ErrInvalidTarget, kind)
}

// Order is what a buyer asked for on the page: a plan, for a submission or a flag of its own.
type Order struct {
	buyer  AccountID
	plan   plans.Plan
	target Target
	// When the buyer asked for the flag to start at once: Stripe keeps it with the payment.
	agreedAt time.Time
}

// Place is the order, once the buyer asked for the flag to start at once and so ended its right to cancel: the EU's
// rule for digital content supplied at once, which Stripe's page does not ask.
func Place(buyer AccountID, plan plans.Plan, target Target, startsNow bool, at time.Time) (Order, error) {
	if !startsNow {
		return Order{}, ErrNoAgreement
	}
	return Order{buyer: buyer, plan: plan, target: target, agreedAt: at}, nil
}

const (
	accountKey    = "account_id"
	submissionKey = "submission_id"
	flagKey       = "flag"
	planKey       = "plan"
	agreedKey     = "starts_now_agreed_at"
)

// Metadata is what Stripe keeps of the order, on the session, and on the subscription that renews it: each payment it
// tells names the order by it.
func (o Order) Metadata() map[string]string {
	metadata := map[string]string{
		accountKey: o.buyer.String(), planKey: string(o.plan), agreedKey: o.agreedAt.UTC().Format(time.RFC3339),
	}
	if o.target.submission != "" {
		metadata[submissionKey] = o.target.submission
	} else {
		metadata[flagKey] = o.target.flag
	}
	return metadata
}

func (o Order) TakeSnapshot() Snapshot {
	return Snapshot{Buyer: o.buyer, Plan: o.plan, Target: o.target, AgreedAt: o.agreedAt}
}

type Snapshot struct {
	Buyer    AccountID
	Plan     plans.Plan
	Target   Target
	AgreedAt time.Time
}

// FromMetadata is the order Stripe tells of. ErrNotOurs for what the shop never sold: the account sells other things.
func FromMetadata(metadata map[string]string) (Snapshot, error) {
	if metadata[planKey] == "" || metadata[accountKey] == "" {
		return Snapshot{}, ErrNotOurs
	}
	buyer, err := ParseAccountID(metadata[accountKey])
	if err != nil {
		return Snapshot{}, err
	}
	plan, err := plans.ParsePlan(metadata[planKey])
	if err != nil {
		return Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}
	target, err := ParseTarget(metadata[submissionKey], metadata[flagKey])
	if err != nil {
		return Snapshot{}, err
	}
	agreedAt, err := time.Parse(time.RFC3339, metadata[agreedKey])
	if err != nil {
		return Snapshot{}, fmt.Errorf("the order's agreement: %w", err)
	}
	return Snapshot{Buyer: buyer, Plan: plan, Target: target, AgreedAt: agreedAt.UTC()}, nil
}
