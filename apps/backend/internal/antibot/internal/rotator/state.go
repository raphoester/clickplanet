package rotator

import (
	"maps"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
)

var _ evidence.Section = (*Watchdog)(nil)

type savedState struct {
	Span     int64
	Accounts []savedAccount
}

type savedAccount struct {
	ID        string
	SignedIn  bool
	Scope     string
	Last      int64
	Countries map[string]int
	Slots     []savedSlot
}

type savedSlot struct {
	At      int64
	Clicked []uint64
}

func (w *Watchdog) Save() ([]byte, error) {
	w.mu.Lock()

	saved := savedState{Span: w.span, Accounts: make([]savedAccount, 0, len(w.accounts))}
	for _, a := range w.accounts {
		slots := make([]savedSlot, 0, len(a.slots))
		for at, clicked := range a.slots {
			slots = append(slots, savedSlot{At: int64(at), Clicked: slices.Clone(clicked)})
		}
		saved.Accounts = append(saved.Accounts, savedAccount{
			ID:        string(a.id),
			SignedIn:  a.signedIn,
			Scope:     a.scope,
			Last:      evidence.Nanos(a.last),
			Countries: maps.Clone(a.countries),
			Slots:     slots,
		})
	}

	w.mu.Unlock()

	return evidence.Encode(saved)
}

// Load keeps nothing saved under another slot length: its marks are seconds of that slot.
func (w *Watchdog) Load(data []byte) error {
	var saved savedState
	if err := evidence.Decode(data, &saved); err != nil {
		return err
	}

	accounts := make(map[accountID]*account, len(saved.Accounts))
	if saved.Span == w.span {
		for _, a := range saved.Accounts {
			loaded := &account{
				id:        accountID(a.ID),
				signedIn:  a.SignedIn,
				scope:     a.Scope,
				last:      evidence.Time(a.Last),
				countries: make(map[string]int, len(a.Countries)),
				slots:     make(map[slot]marks, len(a.Slots)),
			}
			maps.Copy(loaded.countries, a.Countries)
			for _, kept := range a.Slots {
				if len(kept.Clicked) == w.words() {
					loaded.slots[slot(kept.At)] = kept.Clicked
				}
			}
			accounts[loaded.id] = loaded
		}
	}

	w.mu.Lock()
	w.accounts = accounts
	w.mu.Unlock()

	return nil
}

func (w *Watchdog) Forget(before time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.forgetLocked(w.slotOf(before))
}
