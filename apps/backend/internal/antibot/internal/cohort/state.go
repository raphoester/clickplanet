package cohort

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var _ evidence.Section = (*Watchdog)(nil)

// The cached judgement is not saved: a loaded member is judged again on its next click.
type savedMember struct {
	Scope     string
	First     int64
	Last      int64
	Clicks    int
	Countries map[string]int
}

func (w *Watchdog) Save() ([]byte, error) {
	w.mu.Lock()

	saved := make([]savedMember, 0, len(w.members))
	for _, m := range w.members {
		countries := make(map[string]int, len(m.countries))
		for country, n := range m.countries {
			countries[country] = n
		}
		saved = append(saved, savedMember{
			Scope:     m.scope,
			First:     evidence.Nanos(m.first),
			Last:      evidence.Nanos(m.last),
			Clicks:    m.clicks,
			Countries: countries,
		})
	}

	w.mu.Unlock()

	return evidence.Encode(saved)
}

func (w *Watchdog) Load(data []byte) error {
	var saved []savedMember
	if err := evidence.Decode(data, &saved); err != nil {
		return err
	}

	members := make(map[string]*member, len(saved))
	starts := make(map[int64]*cpcolls.Set[string])
	prefixes := make(map[string]*cpcolls.Set[string])

	for _, s := range saved {
		m := &member{
			scope:     s.Scope,
			prefix:    widen(s.Scope, w.config.V4Bits, w.config.V6Bits),
			first:     evidence.Time(s.First),
			last:      evidence.Time(s.Last),
			clicks:    s.Clicks,
			countries: make(map[string]int, len(s.Countries)),
		}
		for country, n := range s.Countries {
			m.countries[country] = n
		}
		members[m.scope] = m

		index(starts, m.first.Unix(), m.scope)
		if m.prefix != "" {
			index(prefixes, m.prefix, m.scope)
		}
	}

	w.mu.Lock()
	w.members, w.starts, w.prefixes = members, starts, prefixes
	w.mu.Unlock()

	return nil
}

func (w *Watchdog) Forget(before time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for scope, m := range w.members {
		if m.last.Before(before) {
			delete(w.members, scope)
			unindex(w.starts, m.first.Unix(), scope)
			if m.prefix != "" {
				unindex(w.prefixes, m.prefix, scope)
			}
		}
	}
}
