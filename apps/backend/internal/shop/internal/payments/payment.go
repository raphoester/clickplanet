package payments

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
)

var (
	ErrUnknownPayment = errors.New("no payment was kept under this id")
	ErrInvalidPayment = errors.New("not a payment of Stripe's")
	ErrInvalidState   = errors.New("a payment kept in no state it can be in")
)

// PaymentID is the Stripe object that took the money: the Checkout Session of a week (cs_), the invoice of a month or
// a year, the first one and each renewal (in_).
type PaymentID string

// IntentID is the Stripe PaymentIntent of a payment (pi_): a refund and a dispute name it.
type IntentID string

var (
	paymentShape = regexp.MustCompile(`^(cs|in)_[A-Za-z0-9_]+$`)
	intentShape  = regexp.MustCompile(`^pi_[A-Za-z0-9_]+$`)
)

func ParsePaymentID(s string) (PaymentID, error) {
	if !paymentShape.MatchString(s) {
		return "", fmt.Errorf("%w: the payment %q", ErrInvalidPayment, s)
	}
	return PaymentID(s), nil
}

func (id PaymentID) String() string {
	return string(id)
}

// GrantID is the payment's name to the flags module, which grants a payment once.
func (id PaymentID) GrantID() string {
	return "stripe:" + string(id)
}

func ParseIntentID(s string) (IntentID, error) {
	if !intentShape.MatchString(s) {
		return "", fmt.Errorf("%w: the intent %q", ErrInvalidPayment, s)
	}
	return IntentID(s), nil
}

func (id IntentID) String() string {
	return string(id)
}

type State string

const (
	Paid State = "paid"
	// Refunded in full, or lost in a dispute: its time was taken back.
	TakenBack State = "taken_back"
)

// Payment is money Stripe took for an order, from its payment to a refund or a dispute lost.
type Payment struct {
	id       PaymentID
	order    orders.Snapshot
	intent   IntentID
	amount   int64
	currency string
	state    State
	paidAt   time.Time
	// When it took its state.
	decidedAt time.Time
}

// Charged is what Stripe tells of a payment: its order, its intent, and how much it took.
type Charged struct {
	ID       PaymentID
	Order    orders.Snapshot
	Intent   IntentID
	Amount   int64
	Currency string
	At       time.Time
}

// Pay keeps a payment, and asks the flags module for the time it bought.
func Pay(charged Charged) (Payment, *flagsv1.GrantTime, error) {
	payment, err := Snapshot{
		ID: charged.ID, Order: charged.Order, Intent: charged.Intent, Amount: charged.Amount, Currency: charged.Currency,
		State: Paid, PaidAt: charged.At, DecidedAt: charged.At,
	}.Restore()
	if err != nil {
		return Payment{}, nil, err
	}
	target := &flagsv1.TimeTarget{}
	if submission := charged.Order.Target.Submission(); submission != "" {
		target.Target = &flagsv1.TimeTarget_SubmissionId{SubmissionId: submission}
	} else {
		target.Target = &flagsv1.TimeTarget_Flag{Flag: charged.Order.Target.Flag()}
	}
	return payment, &flagsv1.GrantTime{
		GrantId: charged.ID.GrantID(), Target: target, Period: periodOf(charged.Order.Plan),
		PaidAt: timestamppb.New(charged.At),
	}, nil
}

func periodOf(plan plans.Plan) flagsv1.TimePeriod {
	switch plan {
	case plans.Week:
		return flagsv1.TimePeriod_TIME_PERIOD_WEEK
	case plans.Month:
		return flagsv1.TimePeriod_TIME_PERIOD_MONTH
	case plans.Year:
		return flagsv1.TimePeriod_TIME_PERIOD_YEAR
	}
	return flagsv1.TimePeriod_TIME_PERIOD_UNSPECIFIED
}

// TakenBack is the payment once it was refunded in full or lost in a dispute, and the flags module told to take its
// time back. A payment taken back already asks nothing more.
func (p Payment) TakenBack(at time.Time) (Payment, *flagsv1.TakeBackTime, bool) {
	if p.state == TakenBack {
		return p, nil, false
	}
	p.state, p.decidedAt = TakenBack, at
	return p, &flagsv1.TakeBackTime{GrantId: p.id.GrantID(), DecidedAt: timestamppb.New(at)}, true
}

func (p Payment) TakeSnapshot() Snapshot {
	return Snapshot{
		ID: p.id, Order: p.order, Intent: p.intent, Amount: p.amount, Currency: p.currency, State: p.state,
		PaidAt: p.paidAt, DecidedAt: p.decidedAt,
	}
}

type Snapshot struct {
	ID     PaymentID
	Order  orders.Snapshot
	Intent IntentID
	// In the currency's smallest unit, taxes included.
	Amount    int64
	Currency  string
	State     State
	PaidAt    time.Time
	DecidedAt time.Time
}

func (s Snapshot) Restore() (Payment, error) {
	if _, err := ParsePaymentID(s.ID.String()); err != nil {
		return Payment{}, err
	}
	if _, err := ParseIntentID(s.Intent.String()); err != nil {
		return Payment{}, fmt.Errorf("the payment %s: %w", s.ID, err)
	}
	if _, err := plans.ParsePlan(string(s.Order.Plan)); err != nil {
		return Payment{}, fmt.Errorf("the payment %s: %w", s.ID, err)
	}
	if _, err := orders.ParseTarget(s.Order.Target.Submission(), s.Order.Target.Flag()); err != nil {
		return Payment{}, fmt.Errorf("the payment %s: %w", s.ID, err)
	}
	switch s.State {
	case Paid, TakenBack:
	default:
		return Payment{}, fmt.Errorf("%w: %s is %q", ErrInvalidState, s.ID, s.State)
	}
	return Payment{
		id: s.ID, order: s.Order, intent: s.Intent, amount: s.Amount, currency: s.Currency, state: s.State,
		paidAt: s.PaidAt, decidedAt: s.DecidedAt,
	}, nil
}
