package metronome

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
)

var (
	_ evidence.Section = (*Watchdog)(nil)
	_ evidence.Resumer = (*Watchdog)(nil)
)

type saved struct {
	Callers  []savedCaller
	Slice    int64
	Spenders []savedSpender
}

type savedCaller struct {
	Scope     string
	LastSeen  int64
	RunStart  int64
	RunClicks int
	Gaps      []int64
	Shape     []int64
	Tries     []int64
}

type savedSpender struct {
	Payer    string
	LastSeen int64
	Slices   []savedSlice
}

type savedSlice struct {
	Index  int64
	Clicks int
}

func (w *Watchdog) Save() ([]byte, error) {
	w.mu.Lock()

	s := saved{
		Callers:  make([]savedCaller, 0, len(w.callers)),
		Slice:    int64(w.config.Stamina.Slice),
		Spenders: make([]savedSpender, 0, len(w.spenders)),
	}
	for scope, c := range w.callers {
		s.Callers = append(s.Callers, savedCaller{
			Scope:     scope,
			LastSeen:  evidence.Nanos(c.lastSeen),
			RunStart:  evidence.Nanos(c.runStart),
			RunClicks: c.runClicks,
			Gaps:      nanos(c.gaps),
			Shape:     nanos(c.shape),
			Tries:     times(c.tries),
		})
	}
	for payer, sp := range w.spenders {
		slices := make([]savedSlice, 0, len(sp.slices))
		for _, counted := range sp.slices {
			slices = append(slices, savedSlice{Index: counted.index, Clicks: counted.clicks})
		}
		s.Spenders = append(s.Spenders, savedSpender{Payer: payer, LastSeen: evidence.Nanos(sp.lastSeen), Slices: slices})
	}

	w.mu.Unlock()

	return evidence.Encode(s)
}

func (w *Watchdog) Load(data []byte) error {
	var s saved
	if err := evidence.Decode(data, &s); err != nil {
		return err
	}

	callers := make(map[string]*caller, len(s.Callers))
	for _, c := range s.Callers {
		callers[c.Scope] = &caller{
			lastSeen:  evidence.Time(c.LastSeen),
			runStart:  evidence.Time(c.RunStart),
			runClicks: c.RunClicks,
			gaps:      durations(c.Gaps, w.capacity()),
			shape:     durations(c.Shape, w.config.Shape.CertainClicks),
			tries:     instants(c.Tries, w.config.Clock.kept()),
		}
	}

	spenders := make(map[string]*spender, len(s.Spenders))
	// Slices counted in another length cannot be read in this one.
	if s.Slice == int64(w.config.Stamina.Slice) {
		for _, sp := range s.Spenders {
			slices := make([]slice, 0, len(sp.Slices))
			for _, counted := range sp.Slices {
				slices = append(slices, slice{index: counted.Index, clicks: counted.Clicks})
			}
			spenders[sp.Payer] = &spender{lastSeen: evidence.Time(sp.LastSeen), slices: slices}
		}
	}

	w.mu.Lock()
	w.callers = callers
	w.spenders = spenders
	w.mu.Unlock()

	return nil
}

// Forget drops a caller silent since before; a run still going is kept whole, however long ago it started.
func nanos(gaps []time.Duration) []int64 {
	saved := make([]int64, 0, len(gaps))
	for _, gap := range gaps {
		saved = append(saved, int64(gap))
	}
	return saved
}

// durations keeps the newest capacity gaps: a section saved under a larger window loads into this one.
func durations(saved []int64, capacity int) []time.Duration {
	if len(saved) > capacity {
		saved = saved[len(saved)-capacity:]
	}
	gaps := make([]time.Duration, 0, len(saved))
	for _, gap := range saved {
		gaps = append(gaps, time.Duration(gap))
	}
	return gaps
}

func times(tries []time.Time) []int64 {
	saved := make([]int64, 0, len(tries))
	for _, at := range tries {
		saved = append(saved, evidence.Nanos(at))
	}
	return saved
}

// instants keeps the newest capacity tries, as durations does.
func instants(saved []int64, capacity int) []time.Time {
	if len(saved) > capacity {
		saved = saved[len(saved)-capacity:]
	}
	tries := make([]time.Time, 0, len(saved))
	for _, at := range saved {
		tries = append(tries, evidence.Time(at))
	}
	return tries
}

func (w *Watchdog) Forget(before time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for scope, c := range w.callers {
		if c.lastSeen.Before(before) {
			delete(w.callers, scope)
		}
	}
	for payer, s := range w.spenders {
		if s.lastSeen.Before(before) {
			delete(w.spenders, payer)
		}
	}
}

func (w *Watchdog) Resume(outage detect.Outage) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.outage = outage
}
