//go:build testing

package bonuses

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// FakeWallet stands for planet's wallet in the offers' tests: it pays each grant once, a refill or a bomb as one held,
// and a stack up to the size it is told (none, no size). It applies none of planet's own rules.
type FakeWallet struct {
	mu       sync.Mutex
	hands    map[Account]Hand
	sizes    map[Kind]int
	paid     *cpcolls.Set[GrantID]
	payments []Payment
	failing  error
	gone     []error
}

func NewFakeWallet() *FakeWallet {
	return &FakeWallet{hands: map[Account]Hand{}, sizes: map[Kind]int{}, paid: cpcolls.NewSet[GrantID]()}
}

// Size caps the stack of a kind, as the wallet's rules would.
func (w *FakeWallet) Size(kind Kind, size int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.sizes[kind] = size
}

func (w *FakeWallet) Keep(account Account, hand Hand) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.hands[account] = hand
}

func (w *FakeWallet) Kept(account Account) Hand {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.hands[account]
}

func (w *FakeWallet) FailWith(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.failing = err
}

// Payments are every payment asked, a grant told again included.
func (w *FakeWallet) Payments() []Payment {
	w.mu.Lock()
	defer w.mu.Unlock()

	return append([]Payment(nil), w.payments...)
}

// Gone is the context's error at each payment: nil while the payment could still run.
func (w *FakeWallet) Gone() []error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return append([]error(nil), w.gone...)
}

func (w *FakeWallet) Pay(ctx context.Context, payment Payment) (Paid, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.payments = append(w.payments, payment)
	w.gone = append(w.gone, ctx.Err())
	if w.failing != nil {
		return Paid{}, w.failing
	}

	hand := w.hands[payment.Account]
	if payment.Account == NoAccount || w.paid.Contains(payment.Grant) {
		return Paid{Hand: hand}, nil
	}
	w.paid.Add(payment.Grant)

	kept := 0
	switch payment.Kind {
	case KindRefill:
		kept, hand.Refill = oneIf(!hand.Refill), true
	case KindBomb:
		kept, hand.Bomb = oneIf(!hand.Bomb), true
	case KindEncloseClicks:
		kept, hand.Enclosures = w.stacked(hand.Enclosures, payment)
	case KindSpreadClicks:
		kept, hand.SpreadClicks = w.stacked(hand.SpreadClicks, payment)
	case KindShields:
		kept, hand.Shields = w.stacked(hand.Shields, payment)
	}
	w.hands[payment.Account] = hand

	return Paid{Kept: kept, Hand: hand}, nil
}

func (w *FakeWallet) Hand(_ context.Context, account Account) (Hand, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.failing != nil {
		return Hand{}, w.failing
	}
	return w.hands[account], nil
}

func (w *FakeWallet) stacked(held int, payment Payment) (int, int) {
	after := held + payment.Amount
	if size, ok := w.sizes[payment.Kind]; ok {
		after = max(held, min(after, size))
	}
	return after - held, after
}

func oneIf(held bool) int {
	if held {
		return 1
	}
	return 0
}
