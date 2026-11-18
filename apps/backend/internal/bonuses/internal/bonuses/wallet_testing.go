//go:build testing

package bonuses

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// Hand is what the fake wallet holds for an account.
type Hand struct {
	Refills int
	Bombs   int

	Enclosures int

	SpreadClicks int

	Shields int

	Viruses int

	Mines int

	Rallies int
}

// FakeWallet stands for planet's wallet in the bonuses module's tests: it pays each grant once, every kind stacked up to
// the size it is told (none, no size), and answers how many it kept. It applies none of planet's own rules.
type FakeWallet struct {
	mu       sync.Mutex
	hands    map[Account]Hand
	sizes    map[Kind]int
	paid     *cpcolls.Set[GrantID]
	payments []Payment
	fills    []Stacks
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

func (w *FakeWallet) Pay(ctx context.Context, payment Payment) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.payments = append(w.payments, payment)
	w.gone = append(w.gone, ctx.Err())
	if w.failing != nil {
		return 0, w.failing
	}

	hand := w.hands[payment.Account]
	if payment.Account == NoAccount || w.paid.Contains(payment.Grant) {
		return 0, nil
	}
	w.paid.Add(payment.Grant)

	count := hand.count(payment.Kind)
	if count == nil {
		return 0, nil
	}
	before := *count
	*count = w.stacked(*count, payment)
	w.hands[payment.Account] = hand

	return *count - before, nil
}

// Fills are every fill asked, a grant told again included.
func (w *FakeWallet) Fills() []Stacks {
	w.mu.Lock()
	defer w.mu.Unlock()

	return append([]Stacks(nil), w.fills...)
}

// Fill tops each kind up to the size it is told: a kind with no size is kept none of.
func (w *FakeWallet) Fill(_ context.Context, stacks Stacks) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.fills = append(w.fills, stacks)
	if w.failing != nil {
		return w.failing
	}
	if stacks.Account == NoAccount || w.paid.Contains(stacks.Grant) {
		return nil
	}
	w.paid.Add(stacks.Grant)

	hand := w.hands[stacks.Account]
	for _, kind := range stacks.Kinds {
		if count := hand.count(kind); count != nil {
			*count = max(*count, w.sizes[kind])
		}
	}
	w.hands[stacks.Account] = hand
	return nil
}

func (h *Hand) count(kind Kind) *int {
	switch kind {
	case KindRefill:
		return &h.Refills
	case KindBomb:
		return &h.Bombs
	case KindEncloseClicks:
		return &h.Enclosures
	case KindSpreadClicks:
		return &h.SpreadClicks
	case KindShields:
		return &h.Shields
	case KindVirus:
		return &h.Viruses
	case KindMine:
		return &h.Mines
	case KindRally:
		return &h.Rallies
	}
	return nil
}

func (w *FakeWallet) stacked(held int, payment Payment) int {
	after := held + payment.Amount
	if size, ok := w.sizes[payment.Kind]; ok {
		after = max(held, min(after, size))
	}
	return after
}
