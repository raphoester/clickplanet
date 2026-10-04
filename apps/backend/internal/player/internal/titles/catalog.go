package titles

import (
	"slices"
	"time"
)

type Catalog struct {
	standalone []Title
	tracks     []Track
}

func NewCatalog() Catalog {
	return CatalogOf([]Title{OG{}}, Conquest{}, Devotion{}, Chatter{})
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
	if !career.Account.Linked {
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
	reconciliation := Reconciliation{Grants: Holdings{}, Revocations: Holdings{}}
	for _, career := range careers {
		account := career.Stats.Account
		earned := c.EarnedBy(career)
		if missing := earned.Without(held[account]); len(missing) > 0 {
			reconciliation.Grants[account] = missing
		}
		if unearned := held[account].Without(earned); len(unearned) > 0 {
			reconciliation.Revocations[account] = unearned
		}
	}
	return reconciliation
}

func (c Catalog) StandingOf(id ID) (Standing, bool) {
	for _, title := range c.standalone {
		if title.ID() == id {
			return Standing{Title: title}, true
		}
	}
	for _, track := range c.tracks {
		ranks := track.Ranks()
		for i, rank := range ranks {
			if rank.ID() == id {
				return Standing{Title: rank, Place: placeOf(track, i)}, true
			}
		}
	}
	return Standing{}, false
}

func (c Catalog) Shown(held IDs) []Standing {
	var shown []Standing
	for _, title := range c.standalone {
		if slices.Contains(held, title.ID()) {
			shown = append(shown, Standing{Title: title})
		}
	}
	for _, track := range c.tracks {
		ranks := track.Ranks()
		for i := len(ranks) - 1; i >= 0; i-- {
			if slices.Contains(held, ranks[i].ID()) {
				shown = append(shown, Standing{Title: ranks[i], Place: placeOf(track, i)})
				break
			}
		}
	}
	return shown
}

type Step struct {
	Standing  Standing
	Threshold uint64
	Earned    bool
}

type TrackProgress struct {
	ID       TrackID
	Name     string
	Progress uint64
	Steps    []Step
}

func (c Catalog) Progress(career Career, held IDs) []TrackProgress {
	progress := make([]TrackProgress, 0, len(c.tracks))
	for _, track := range c.tracks {
		ranks := track.Ranks()
		steps := make([]Step, 0, len(ranks))
		for i, rank := range ranks {
			steps = append(steps, Step{
				Standing:  Standing{Title: rank, Place: placeOf(track, i)},
				Threshold: rank.Threshold(),
				Earned:    slices.Contains(held, rank.ID()),
			})
		}
		progress = append(progress, TrackProgress{ID: track.ID(), Name: track.Name(), Progress: track.Progress(career), Steps: steps})
	}
	return progress
}

func placeOf(track Track, index int) Place {
	return Place{Track: track.ID(), TrackName: track.Name(), Number: index + 1, Count: len(track.Ranks())}
}

var ogCutoff = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

type OG struct{}

func (OG) ID() ID { return "og" }

func (OG) Name() string { return "OG" }

func (OG) EarnedBy(career Career) bool {
	return !career.Account.CreatedAt.IsZero() && career.Account.CreatedAt.Before(ogCutoff)
}

type Settler struct{}

func (Settler) ID() ID { return "settler" }

func (Settler) Name() string { return "Settler" }

func (Settler) Threshold() uint64 { return 100 }

func (s Settler) EarnedBy(career Career) bool { return career.Stats.TilesTaken >= s.Threshold() }

type Raider struct{}

func (Raider) ID() ID { return "raider" }

func (Raider) Name() string { return "Raider" }

func (Raider) Threshold() uint64 { return 1_000 }

func (r Raider) EarnedBy(career Career) bool { return career.Stats.TilesTaken >= r.Threshold() }

type Warlord struct{}

func (Warlord) ID() ID { return "warlord" }

func (Warlord) Name() string { return "Warlord" }

func (Warlord) Threshold() uint64 { return 10_000 }

func (w Warlord) EarnedBy(career Career) bool { return career.Stats.TilesTaken >= w.Threshold() }

type Conqueror struct{}

func (Conqueror) ID() ID { return "conqueror" }

func (Conqueror) Name() string { return "Conqueror" }

func (Conqueror) Threshold() uint64 { return 100_000 }

func (c Conqueror) EarnedBy(career Career) bool { return career.Stats.TilesTaken >= c.Threshold() }

type Warmaster struct{}

func (Warmaster) ID() ID { return "warmaster" }

func (Warmaster) Name() string { return "Warmaster" }

func (Warmaster) Threshold() uint64 { return 1_000_000 }

func (w Warmaster) EarnedBy(career Career) bool { return career.Stats.TilesTaken >= w.Threshold() }

type Loyal struct{}

func (Loyal) ID() ID { return "loyal" }

func (Loyal) Name() string { return "Loyal" }

func (Loyal) Threshold() uint64 { return 7 }

func (l Loyal) EarnedBy(career Career) bool { return uint64(career.Stats.StreakBest) >= l.Threshold() }

type Devoted struct{}

func (Devoted) ID() ID { return "devoted" }

func (Devoted) Name() string { return "Devoted" }

func (Devoted) Threshold() uint64 { return 30 }

func (d Devoted) EarnedBy(career Career) bool {
	return uint64(career.Stats.StreakBest) >= d.Threshold()
}

type Unbroken struct{}

func (Unbroken) ID() ID { return "unbroken" }

func (Unbroken) Name() string { return "Unbroken" }

func (Unbroken) Threshold() uint64 { return 100 }

func (u Unbroken) EarnedBy(career Career) bool {
	return uint64(career.Stats.StreakBest) >= u.Threshold()
}

type Talker struct{}

func (Talker) ID() ID { return "talker" }

func (Talker) Name() string { return "Talker" }

func (Talker) Threshold() uint64 { return 100 }

func (t Talker) EarnedBy(career Career) bool { return career.Stats.MessagesSent >= t.Threshold() }

type Chatterbox struct{}

func (Chatterbox) ID() ID { return "chatterbox" }

func (Chatterbox) Name() string { return "Chatterbox" }

func (Chatterbox) Threshold() uint64 { return 1_000 }

func (c Chatterbox) EarnedBy(career Career) bool { return career.Stats.MessagesSent >= c.Threshold() }

type Socialite struct{}

func (Socialite) ID() ID { return "socialite" }

func (Socialite) Name() string { return "Socialite" }

func (Socialite) Threshold() uint64 { return 10_000 }

func (s Socialite) EarnedBy(career Career) bool { return career.Stats.MessagesSent >= s.Threshold() }

type Icon struct{}

func (Icon) ID() ID { return "icon" }

func (Icon) Name() string { return "Icon" }

func (Icon) Threshold() uint64 { return 100_000 }

func (i Icon) EarnedBy(career Career) bool { return career.Stats.MessagesSent >= i.Threshold() }
