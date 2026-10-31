package titles

import (
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
)

type Catalog struct {
	standalone []Title
	tracks     []Track
}

func NewCatalog() Catalog {
	return CatalogOf(
		[]Title{OG{}, Season0Gold{}, Season0Silver{}, Season0Bronze{}, Season0Veteran{}},
		Conquest{}, Devotion{}, Chatter{}, Defense{}, Bombs{}, Loot{}, Quiz{}, Missions{},
	)
}

func CatalogOf(standalone []Title, tracks ...Track) Catalog {
	return Catalog{standalone: standalone, tracks: tracks}
}

func (c Catalog) Titles() []Title {
	titles := slices.Clone(c.standalone)
	for _, track := range c.tracks {
		for _, rank := range track.Ranks() {
			titles = append(titles, rank)
		}
	}
	return titles
}

func (c Catalog) EarnedBy(career Career) IDs {
	if !career.account.Linked() {
		return nil
	}

	var earned IDs
	for _, title := range c.Titles() {
		if title.EarnedBy(career) {
			earned = append(earned, title.ID())
		}
	}
	return earned
}

func (c Catalog) ReconciliationOf(careers []Career, held Holdings) Reconciliation {
	reconciliation := Reconciliation{grants: Holdings{}, revocations: Holdings{}}
	for _, career := range careers {
		account := career.stats.Account()
		earned := c.EarnedBy(career)
		if missing := earned.Without(held[account]); len(missing) > 0 {
			reconciliation.grants[account] = missing
		}
		if unearned := held[account].Without(earned); len(unearned) > 0 {
			reconciliation.revocations[account] = unearned
		}
	}
	return reconciliation
}

func (c Catalog) StandingOf(id ID) (Standing, bool) {
	for _, title := range c.standalone {
		if title.ID() == id {
			return Standing{title: title}, true
		}
	}
	for _, track := range c.tracks {
		ranks := track.Ranks()
		for i, rank := range ranks {
			if rank.ID() == id {
				return Standing{title: rank, place: placeOf(track, i)}, true
			}
		}
	}
	return Standing{}, false
}

func (c Catalog) Shown(held IDs) []Standing {
	var shown []Standing
	for _, title := range c.standalone {
		if slices.Contains(held, title.ID()) {
			shown = append(shown, Standing{title: title})
		}
	}
	for _, track := range c.tracks {
		ranks := track.Ranks()
		for i := len(ranks) - 1; i >= 0; i-- {
			if slices.Contains(held, ranks[i].ID()) {
				shown = append(shown, Standing{title: ranks[i], place: placeOf(track, i)})
				break
			}
		}
	}
	return shown
}

type Step struct {
	standing  Standing
	threshold uint64
	earned    bool
}

func (s Step) Standing() Standing { return s.standing }

func (s Step) Threshold() uint64 { return s.threshold }

func (s Step) Earned() bool { return s.earned }

type TrackProgress struct {
	id       TrackID
	name     string
	progress uint64
	steps    []Step
}

func (t TrackProgress) ID() TrackID { return t.id }

func (t TrackProgress) Name() string { return t.name }

func (t TrackProgress) Progress() uint64 { return t.progress }

func (t TrackProgress) Steps() []Step { return t.steps }

func (c Catalog) Progress(career Career, held IDs) []TrackProgress {
	progress := make([]TrackProgress, 0, len(c.tracks))
	for _, track := range c.tracks {
		ranks := track.Ranks()
		steps := make([]Step, 0, len(ranks))
		for i, rank := range ranks {
			steps = append(steps, Step{
				standing:  Standing{title: rank, place: placeOf(track, i)},
				threshold: rank.Threshold(),
				earned:    slices.Contains(held, rank.ID()),
			})
		}
		progress = append(progress, TrackProgress{id: track.ID(), name: track.Name(), progress: track.Progress(career), steps: steps})
	}
	return progress
}

func placeOf(track Track, index int) Place {
	return Place{track: track.ID(), trackName: track.Name(), number: index + 1, count: len(track.Ranks())}
}

var ogCutoff = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

type OG struct{}

func (OG) ID() ID { return "og" }

func (OG) Name() string { return "OG" }

func (OG) EarnedBy(career Career) bool {
	return !career.account.CreatedAt().IsZero() && career.account.CreatedAt().Before(ogCutoff)
}

const seasonZero finales.Season = 0

type Season0Gold struct{}

func (Season0Gold) ID() ID { return "season_0_gold" }

func (Season0Gold) Name() string { return "Season 0 Gold" }

func (Season0Gold) EarnedBy(career Career) bool { return career.fights.PlacedIn(seasonZero, 1) }

type Season0Silver struct{}

func (Season0Silver) ID() ID { return "season_0_silver" }

func (Season0Silver) Name() string { return "Season 0 Silver" }

func (Season0Silver) EarnedBy(career Career) bool { return career.fights.PlacedIn(seasonZero, 2) }

type Season0Bronze struct{}

func (Season0Bronze) ID() ID { return "season_0_bronze" }

func (Season0Bronze) Name() string { return "Season 0 Bronze" }

func (Season0Bronze) EarnedBy(career Career) bool { return career.fights.PlacedIn(seasonZero, 3) }

type Season0Veteran struct{}

func (Season0Veteran) ID() ID { return "season_0_veteran" }

func (Season0Veteran) Name() string { return "Season 0 Veteran" }

func (Season0Veteran) EarnedBy(career Career) bool { return career.fights.FoughtIn(seasonZero) }

type Settler struct{}

func (Settler) ID() ID { return "settler" }

func (Settler) Name() string { return "Settler" }

func (Settler) Threshold() uint64 { return 100 }

func (s Settler) EarnedBy(career Career) bool { return career.stats.TilesTaken() >= s.Threshold() }

type Raider struct{}

func (Raider) ID() ID { return "raider" }

func (Raider) Name() string { return "Raider" }

func (Raider) Threshold() uint64 { return 1_000 }

func (r Raider) EarnedBy(career Career) bool { return career.stats.TilesTaken() >= r.Threshold() }

type Warlord struct{}

func (Warlord) ID() ID { return "warlord" }

func (Warlord) Name() string { return "Warlord" }

func (Warlord) Threshold() uint64 { return 10_000 }

func (w Warlord) EarnedBy(career Career) bool { return career.stats.TilesTaken() >= w.Threshold() }

type Conqueror struct{}

func (Conqueror) ID() ID { return "conqueror" }

func (Conqueror) Name() string { return "Conqueror" }

func (Conqueror) Threshold() uint64 { return 100_000 }

func (c Conqueror) EarnedBy(career Career) bool { return career.stats.TilesTaken() >= c.Threshold() }

type Warmaster struct{}

func (Warmaster) ID() ID { return "warmaster" }

func (Warmaster) Name() string { return "Warmaster" }

func (Warmaster) Threshold() uint64 { return 1_000_000 }

func (w Warmaster) EarnedBy(career Career) bool { return career.stats.TilesTaken() >= w.Threshold() }

type Loyal struct{}

func (Loyal) ID() ID { return "loyal" }

func (Loyal) Name() string { return "Loyal" }

func (Loyal) Threshold() uint64 { return 7 }

func (l Loyal) EarnedBy(career Career) bool {
	return uint64(career.stats.StreakBest()) >= l.Threshold()
}

type Devoted struct{}

func (Devoted) ID() ID { return "devoted" }

func (Devoted) Name() string { return "Devoted" }

func (Devoted) Threshold() uint64 { return 30 }

func (d Devoted) EarnedBy(career Career) bool {
	return uint64(career.stats.StreakBest()) >= d.Threshold()
}

type Unbroken struct{}

func (Unbroken) ID() ID { return "unbroken" }

func (Unbroken) Name() string { return "Unbroken" }

func (Unbroken) Threshold() uint64 { return 100 }

func (u Unbroken) EarnedBy(career Career) bool {
	return uint64(career.stats.StreakBest()) >= u.Threshold()
}

type Talker struct{}

func (Talker) ID() ID { return "talker" }

func (Talker) Name() string { return "Talker" }

func (Talker) Threshold() uint64 { return 10 }

func (t Talker) EarnedBy(career Career) bool { return career.stats.MessagesSent() >= t.Threshold() }

type Chatterbox struct{}

func (Chatterbox) ID() ID { return "chatterbox" }

func (Chatterbox) Name() string { return "Chatterbox" }

func (Chatterbox) Threshold() uint64 { return 30 }

func (c Chatterbox) EarnedBy(career Career) bool { return career.stats.MessagesSent() >= c.Threshold() }

type Socialite struct{}

func (Socialite) ID() ID { return "socialite" }

func (Socialite) Name() string { return "Socialite" }

func (Socialite) Threshold() uint64 { return 100 }

func (s Socialite) EarnedBy(career Career) bool { return career.stats.MessagesSent() >= s.Threshold() }

type Icon struct{}

func (Icon) ID() ID { return "icon" }

func (Icon) Name() string { return "Icon" }

func (Icon) Threshold() uint64 { return 1_000 }

func (i Icon) EarnedBy(career Career) bool { return career.stats.MessagesSent() >= i.Threshold() }

type Guard struct{}

func (Guard) ID() ID { return "guard" }

func (Guard) Name() string { return "Guard" }

func (Guard) Threshold() uint64 { return 100 }

func (g Guard) EarnedBy(career Career) bool { return career.stats.TilesFortified() >= g.Threshold() }

type Defender struct{}

func (Defender) ID() ID { return "defender" }

func (Defender) Name() string { return "Defender" }

func (Defender) Threshold() uint64 { return 1_000 }

func (d Defender) EarnedBy(career Career) bool { return career.stats.TilesFortified() >= d.Threshold() }

type Protector struct{}

func (Protector) ID() ID { return "protector" }

func (Protector) Name() string { return "Protector" }

func (Protector) Threshold() uint64 { return 10_000 }

func (p Protector) EarnedBy(career Career) bool {
	return career.stats.TilesFortified() >= p.Threshold()
}

type Warden struct{}

func (Warden) ID() ID { return "warden" }

func (Warden) Name() string { return "Warden" }

func (Warden) Threshold() uint64 { return 100_000 }

func (w Warden) EarnedBy(career Career) bool {
	return career.stats.TilesFortified() >= w.Threshold()
}

type Bomber struct{}

func (Bomber) ID() ID { return "bomber" }

func (Bomber) Name() string { return "Bomber" }

func (Bomber) Threshold() uint64 { return 1 }

func (b Bomber) EarnedBy(career Career) bool { return career.stats.BombsDropped() >= b.Threshold() }

type Blaster struct{}

func (Blaster) ID() ID { return "blaster" }

func (Blaster) Name() string { return "Blaster" }

func (Blaster) Threshold() uint64 { return 10 }

func (b Blaster) EarnedBy(career Career) bool { return career.stats.BombsDropped() >= b.Threshold() }

type Wrecker struct{}

func (Wrecker) ID() ID { return "wrecker" }

func (Wrecker) Name() string { return "Wrecker" }

func (Wrecker) Threshold() uint64 { return 50 }

func (w Wrecker) EarnedBy(career Career) bool { return career.stats.BombsDropped() >= w.Threshold() }

type Destroyer struct{}

func (Destroyer) ID() ID { return "destroyer" }

func (Destroyer) Name() string { return "Destroyer" }

func (Destroyer) Threshold() uint64 { return 250 }

func (d Destroyer) EarnedBy(career Career) bool { return career.stats.BombsDropped() >= d.Threshold() }

type Looter struct{}

func (Looter) ID() ID { return "looter" }

func (Looter) Name() string { return "Looter" }

func (Looter) Threshold() uint64 { return 10 }

func (l Looter) EarnedBy(career Career) bool {
	return career.stats.BonusesClaimed() >= l.Threshold()
}

type Hunter struct{}

func (Hunter) ID() ID { return "hunter" }

func (Hunter) Name() string { return "Hunter" }

func (Hunter) Threshold() uint64 { return 100 }

func (h Hunter) EarnedBy(career Career) bool { return career.stats.BonusesClaimed() >= h.Threshold() }

type Pirate struct{}

func (Pirate) ID() ID { return "pirate" }

func (Pirate) Name() string { return "Pirate" }

func (Pirate) Threshold() uint64 { return 500 }

func (p Pirate) EarnedBy(career Career) bool {
	return career.stats.BonusesClaimed() >= p.Threshold()
}

type Captain struct{}

func (Captain) ID() ID { return "captain" }

func (Captain) Name() string { return "Captain" }

func (Captain) Threshold() uint64 { return 2_500 }

func (c Captain) EarnedBy(career Career) bool {
	return career.stats.BonusesClaimed() >= c.Threshold()
}

type Rookie struct{}

func (Rookie) ID() ID { return "rookie" }

func (Rookie) Name() string { return "Rookie" }

func (Rookie) Threshold() uint64 { return 1 }

func (r Rookie) EarnedBy(career Career) bool { return career.stats.MissionsDone() >= r.Threshold() }

type Agent struct{}

func (Agent) ID() ID { return "agent" }

func (Agent) Name() string { return "Agent" }

func (Agent) Threshold() uint64 { return 25 }

func (a Agent) EarnedBy(career Career) bool { return career.stats.MissionsDone() >= a.Threshold() }

type Operative struct{}

func (Operative) ID() ID { return "operative" }

func (Operative) Name() string { return "Operative" }

func (Operative) Threshold() uint64 { return 100 }

func (o Operative) EarnedBy(career Career) bool { return career.stats.MissionsDone() >= o.Threshold() }

type Spymaster struct{}

func (Spymaster) ID() ID { return "spymaster" }

func (Spymaster) Name() string { return "Spymaster" }

func (Spymaster) Threshold() uint64 { return 500 }

func (s Spymaster) EarnedBy(career Career) bool { return career.stats.MissionsDone() >= s.Threshold() }

type Student struct{}

func (Student) ID() ID { return "student" }

func (Student) Name() string { return "Student" }

func (Student) Threshold() uint64 { return 10 }

func (s Student) EarnedBy(career Career) bool { return career.stats.RightAnswers() >= s.Threshold() }

type Expert struct{}

func (Expert) ID() ID { return "expert" }

func (Expert) Name() string { return "Expert" }

func (Expert) Threshold() uint64 { return 50 }

func (e Expert) EarnedBy(career Career) bool { return career.stats.RightAnswers() >= e.Threshold() }

type Professor struct{}

func (Professor) ID() ID { return "professor" }

func (Professor) Name() string { return "Professor" }

func (Professor) Threshold() uint64 { return 250 }

func (p Professor) EarnedBy(career Career) bool {
	return career.stats.RightAnswers() >= p.Threshold()
}

type Genius struct{}

func (Genius) ID() ID { return "genius" }

func (Genius) Name() string { return "Genius" }

func (Genius) Threshold() uint64 { return 1_000 }

func (g Genius) EarnedBy(career Career) bool { return career.stats.RightAnswers() >= g.Threshold() }
