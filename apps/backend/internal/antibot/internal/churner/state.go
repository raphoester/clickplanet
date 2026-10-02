package churner

import (
	"maps"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
)

var _ evidence.Section = (*Watchdog)(nil)

type savedAccount struct {
	ID        string
	Scope     string
	First     int64
	Last      int64
	Clicks    int
	Countries map[string]int
}

func (w *Watchdog) Save() ([]byte, error) {
	w.mu.Lock()

	saved := make([]savedAccount, 0, len(w.accounts))
	for _, a := range w.accounts {
		saved = append(saved, savedAccount{
			ID:        a.id,
			Scope:     a.scope,
			First:     evidence.Nanos(a.first),
			Last:      evidence.Nanos(a.last),
			Clicks:    a.clicks,
			Countries: maps.Clone(a.countries),
		})
	}

	w.mu.Unlock()

	return evidence.Encode(saved)
}

func (w *Watchdog) Load(data []byte) error {
	var saved []savedAccount
	if err := evidence.Decode(data, &saved); err != nil {
		return err
	}

	loaded := New(w.config, w.clock, w.onAccounts, w.onLinks)
	for _, s := range saved {
		countries := make(map[string]int, len(s.Countries))
		maps.Copy(countries, s.Countries)

		loaded.indexLocked(&account{
			id:        s.ID,
			scope:     s.Scope,
			prefix:    detect.WiderPrefix(s.Scope, w.config.Relay.V4Bits, w.config.Relay.V6Bits),
			first:     evidence.Time(s.First),
			last:      evidence.Time(s.Last),
			clicks:    s.Clicks,
			countries: countries,
		})
	}

	w.mu.Lock()
	w.accounts, w.born, w.prefixes = loaded.accounts, loaded.born, loaded.prefixes
	w.mu.Unlock()

	return nil
}

func (w *Watchdog) Forget(before time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, a := range w.accounts {
		if a.last.Before(before) {
			w.dropLocked(a)
		}
	}
}
