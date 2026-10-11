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
	Payers   []savedPayer
	Slice    int64
	Spenders []savedSpender
}

type savedPayer struct {
	Payer     string
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
		Payers:   make([]savedPayer, 0, len(w.payers)),
		Slice:    int64(w.config.Stamina.Slice),
		Spenders: make([]savedSpender, 0, len(w.spenders)),
	}
	for key, p := range w.payers {
		s.Payers = append(s.Payers, savedPayer{
			Payer:     key,
			LastSeen:  evidence.Nanos(p.lastSeen),
			RunStart:  evidence.Nanos(p.runStart),
			RunClicks: p.runClicks,
			Gaps:      nanos(p.gaps),
			Shape:     nanos(p.shape),
			Tries:     times(p.tries),
		})
	}
	for key, sp := range w.spenders {
		slices := make([]savedSlice, 0, len(sp.slices))
		for _, counted := range sp.slices {
			slices = append(slices, savedSlice{Index: counted.index, Clicks: counted.clicks})
		}
		s.Spenders = append(s.Spenders, savedSpender{Payer: key, LastSeen: evidence.Nanos(sp.lastSeen), Slices: slices})
	}

	w.mu.Unlock()

	return evidence.Encode(s)
}

func (w *Watchdog) Load(data []byte) error {
	var s saved
	if err := evidence.Decode(data, &s); err != nil {
		return err
	}

	payers := make(map[string]*payer, len(s.Payers))
	for _, p := range s.Payers {
		payers[p.Payer] = &payer{
			lastSeen:  evidence.Time(p.LastSeen),
			runStart:  evidence.Time(p.RunStart),
			runClicks: p.RunClicks,
			gaps:      durations(p.Gaps, w.capacity()),
			shape:     durations(p.Shape, w.config.Shape.CertainClicks),
			tries:     instants(p.Tries, w.config.Clock.kept()),
		}
	}

	spenders := make(map[string]*spender, len(s.Spenders))
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
	w.payers = payers
	w.spenders = spenders
	w.mu.Unlock()

	return nil
}

func nanos(gaps []time.Duration) []int64 {
	saved := make([]int64, 0, len(gaps))
	for _, gap := range gaps {
		saved = append(saved, int64(gap))
	}
	return saved
}

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

	for key, p := range w.payers {
		if p.lastSeen.Before(before) {
			delete(w.payers, key)
		}
	}
	for key, s := range w.spenders {
		if s.lastSeen.Before(before) {
			delete(w.spenders, key)
		}
	}
}

func (w *Watchdog) Resume(outage detect.Outage) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.outage = outage
}
