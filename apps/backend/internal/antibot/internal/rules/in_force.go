package rules

import (
	"sync/atomic"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/catcher"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/churner"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/cohort"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/defender"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/hopper"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/jury"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rotator"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/scraper"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/sequencer"
)

// InForce is the set the antibot judges with, read on every judgement. Nothing reads it before the boot enforced a set.
type InForce struct {
	set atomic.Pointer[Set]
}

func NewInForce() *InForce {
	return &InForce{}
}

func (f *InForce) Enforce(set Set) {
	f.set.Store(&set)
}

func (f *InForce) Name() SetName { return f.set.Load().name }

func (f *InForce) Bans() bans.Rules { return f.set.Load().bans }

func (f *InForce) Jury() jury.Rules { return f.set.Load().jury }

func (f *InForce) Retaker() retaker.Rules { return f.set.Load().retaker }

func (f *InForce) Sequencer() sequencer.Rules { return f.set.Load().sequencer }

func (f *InForce) Metronome() metronome.Rules { return f.set.Load().metronome }

func (f *InForce) Defender() defender.Rules { return f.set.Load().defender }

func (f *InForce) Catcher() catcher.Rules { return f.set.Load().catcher }

func (f *InForce) Cohort() cohort.Rules { return f.set.Load().cohort }

func (f *InForce) Scraper() scraper.Rules { return f.set.Load().scraper }

func (f *InForce) Churner() churner.Rules { return f.set.Load().churner }

func (f *InForce) Hopper() hopper.Rules { return f.set.Load().hopper }

func (f *InForce) Rotator() rotator.Rules { return f.set.Load().rotator }
