//go:build testing

package bonuses

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// Hand is what the fake wallet holds for an account.
type Hand struct {
	Refill bool
	Bomb   bool

	Enclosures int

	SpreadClicks int

	Shields int

	Virus bool

	Mines int

	Rallies int
}

// FakeWallet stands for planet's wallet in the bonuses module's tests: it pays each grant once, a refill or a bomb as one
// held, and a stack up to the size it is told (none, no size), and says a hand full where it holds that much. It applies
// none of planet's own rules.
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

func (w *FakeWallet) Pay(ctx context.Context, payment Payment) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.payments = append(w.payments, payment)
	w.gone = append(w.gone, ctx.Err())
	if w.failing != nil {
		return w.failing
	}

	hand := w.hands[payment.Account]
	if payment.Account == NoAccount || w.paid.Contains(payment.Grant) {
		return nil
	}
	w.paid.Add(payment.Grant)

	switch payment.Kind {
	case KindRefill:
		hand.Refill = true
	case KindBomb:
		hand.Bomb = true
	case KindEncloseClicks:
		hand.Enclosures = w.stacked(hand.Enclosures, payment)
	case KindSpreadClicks:
		hand.SpreadClicks = w.stacked(hand.SpreadClicks, payment)
	case KindShields:
		hand.Shields = w.stacked(hand.Shields, payment)
	case KindVirus:
		hand.Virus = true
	case KindMine:
		hand.Mines = w.stacked(hand.Mines, payment)
	case KindRally:
		hand.Rallies = w.stacked(hand.Rallies, payment)
	}
	w.hands[payment.Account] = hand

	return nil
}

// Full is every kind a hand holds as many of as its size, or one of where one is the most.
func (w *FakeWallet) Full(_ context.Context, accounts []Account) (map[Account][]Kind, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.failing != nil {
		return nil, w.failing
	}
	full := make(map[Account][]Kind, len(accounts))
	for _, account := range accounts {
		hand := w.hands[account]
		if hand.Refill {
			full[account] = append(full[account], KindRefill)
		}
		if hand.Bomb {
			full[account] = append(full[account], KindBomb)
		}
		if hand.Virus {
			full[account] = append(full[account], KindVirus)
		}
		for _, stack := range []struct {
			kind Kind
			held int
		}{
			{KindEncloseClicks, hand.Enclosures},
			{KindSpreadClicks, hand.SpreadClicks},
			{KindShields, hand.Shields},
			{KindMine, hand.Mines},
			{KindRally, hand.Rallies},
		} {
			if size, ok := w.sizes[stack.kind]; ok && stack.held >= size {
				full[account] = append(full[account], stack.kind)
			}
		}
	}
	return full, nil
}

func (w *FakeWallet) stacked(held int, payment Payment) int {
	after := held + payment.Amount
	if size, ok := w.sizes[payment.Kind]; ok {
		after = max(held, min(after, size))
	}
	return after
}
