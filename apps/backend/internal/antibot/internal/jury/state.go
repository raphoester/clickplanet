package jury

import (
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
)

var (
	_ evidence.Section = (*Jury)(nil)
	_ evidence.Resumer = (*Jury)(nil)
)

type savedCaller struct {
	Scope      string
	FirstSeen  int64
	LastSeen   int64
	LongestGap int64
	Clicks     int
	Countries  map[string]int
	Tiles      []uint32
	Opinions   []savedOpinion
}

type savedOpinion struct {
	Watchdog string
	Verdict  uint8
	Rule     string
	Fields   []savedField
	At       int64
}

// A field's value is only ever rendered with %v, so it is saved already rendered.
type savedField struct {
	Key   string
	Value string
}

func (j *Jury) Name() string { return "jury" }

func (j *Jury) Save() ([]byte, error) {
	j.mu.Lock()

	saved := make([]savedCaller, 0, len(j.callers))
	for scope, c := range j.callers {
		opinions := make([]savedOpinion, 0, len(c.opinions))
		for _, o := range c.opinions {
			fields := make([]savedField, 0, len(o.Evidence.Fields))
			for _, field := range o.Evidence.Fields {
				fields = append(fields, savedField{Key: field.Key, Value: fmt.Sprint(field.Value)})
			}
			opinions = append(opinions, savedOpinion{
				Watchdog: o.Watchdog,
				Verdict:  uint8(o.Verdict),
				Rule:     o.Evidence.Rule,
				Fields:   fields,
				At:       evidence.Nanos(o.At),
			})
		}

		countries := make(map[string]int, len(c.countries))
		for country, n := range c.countries {
			countries[country] = n
		}

		saved = append(saved, savedCaller{
			Scope:      scope,
			FirstSeen:  evidence.Nanos(c.firstSeen),
			LastSeen:   evidence.Nanos(c.lastSeen),
			LongestGap: int64(c.longestGap),
			Clicks:     c.clicks,
			Countries:  countries,
			Tiles:      append([]uint32(nil), c.tiles...),
			Opinions:   opinions,
		})
	}

	j.mu.Unlock()

	return evidence.Encode(saved)
}

func (j *Jury) Load(data []byte) error {
	var saved []savedCaller
	if err := evidence.Decode(data, &saved); err != nil {
		return err
	}

	callers := make(map[string]*caller, len(saved))
	for _, c := range saved {
		loaded := &caller{
			firstSeen:  evidence.Time(c.FirstSeen),
			lastSeen:   evidence.Time(c.LastSeen),
			longestGap: time.Duration(c.LongestGap),
			clicks:     c.Clicks,
			countries:  make(map[string]int, len(c.Countries)),
			tiles:      c.Tiles,
			opinions:   make(map[string]detect.Opinion, len(c.Opinions)),
		}
		for country, n := range c.Countries {
			loaded.countries[country] = n
		}
		if len(loaded.tiles) > keptTiles {
			loaded.tiles = loaded.tiles[len(loaded.tiles)-keptTiles:]
		}
		for _, o := range c.Opinions {
			fields := make([]detect.Field, 0, len(o.Fields))
			for _, field := range o.Fields {
				fields = append(fields, detect.Field{Key: field.Key, Value: field.Value})
			}
			loaded.opinions[o.Watchdog] = detect.Opinion{
				Watchdog: o.Watchdog,
				Verdict:  detect.Verdict(o.Verdict),
				Evidence: detect.Evidence{Rule: o.Rule, Fields: fields},
				At:       evidence.Time(o.At),
			}
		}
		callers[c.Scope] = loaded
	}

	j.mu.Lock()
	j.callers = callers
	j.mu.Unlock()

	return nil
}

// Forget drops a caller silent since before, and any opinion given before it.
func (j *Jury) Forget(before time.Time) {
	j.mu.Lock()
	defer j.mu.Unlock()

	for scope, c := range j.callers {
		if c.lastSeen.Before(before) {
			delete(j.callers, scope)
			continue
		}
		for watchdog, o := range c.opinions {
			if o.At.Before(before) {
				delete(c.opinions, watchdog)
			}
		}
	}
}

func (j *Jury) Resume(outage detect.Outage) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.outage = outage
}
