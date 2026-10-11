package rotator

import (
	"maps"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
)

var _ evidence.Section = (*Watchdog)(nil)

type savedState struct {
	Span   int64
	Scopes []savedScope
}

type savedScope struct {
	Scope    string
	Last     int64
	Accounts []savedAccount
}

type savedAccount struct {
	ID        string
	SignedIn  bool
	Countries map[string]int
	Slots     []savedSlot
}

type savedSlot struct {
	At      int64
	Clicked []uint64
}

func (w *Watchdog) Save() ([]byte, error) {
	w.mu.Lock()

	saved := savedState{Span: w.span, Scopes: make([]savedScope, 0, len(w.scopes))}
	for scope, n := range w.scopes {
		accounts := make([]savedAccount, 0, len(n.accounts))
		for _, a := range n.accounts {
			slots := make([]savedSlot, 0, len(a.slots))
			for at, clicked := range a.slots {
				slots = append(slots, savedSlot{At: int64(at), Clicked: slices.Clone(clicked)})
			}
			accounts = append(accounts, savedAccount{
				ID:        string(a.id),
				SignedIn:  a.signedIn,
				Countries: maps.Clone(a.countries),
				Slots:     slots,
			})
		}
		saved.Scopes = append(saved.Scopes, savedScope{Scope: string(scope), Last: evidence.Nanos(n.last), Accounts: accounts})
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

	scopes := make(map[scopeID]*network, len(saved.Scopes))
	if saved.Span == w.span {
		for _, s := range saved.Scopes {
			n := &network{accounts: make(map[accountID]*account, len(s.Accounts)), last: evidence.Time(s.Last)}
			for _, a := range s.Accounts {
				loaded := &account{
					id:        accountID(a.ID),
					signedIn:  a.SignedIn,
					countries: make(map[string]int, len(a.Countries)),
					slots:     make(map[slot]marks, len(a.Slots)),
				}
				maps.Copy(loaded.countries, a.Countries)
				for _, kept := range a.Slots {
					if len(kept.Clicked) == w.words() {
						loaded.slots[slot(kept.At)] = kept.Clicked
					}
				}
				n.accounts[loaded.id] = loaded
			}
			scopes[scopeID(s.Scope)] = n
		}
	}

	w.mu.Lock()
	w.scopes = scopes
	w.mu.Unlock()

	return nil
}

func (w *Watchdog) Forget(before time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.forgetLocked(w.slotOf(before))
}
