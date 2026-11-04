package titles

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type TrackID string

type Rank interface {
	Title
	Threshold() uint64
}

type Track interface {
	ID() TrackID
	Name() string
	Ranks() []Rank
	Progress(career Career) uint64
	Feat(threshold uint64) string
}

type Conquest struct{}

func (Conquest) ID() TrackID { return "conquest" }

func (Conquest) Name() string { return "Conquest" }

func (Conquest) Ranks() []Rank {
	return []Rank{Settler{}, Raider{}, Warlord{}, Conqueror{}, Warmaster{}}
}

func (Conquest) Progress(career Career) uint64 { return career.Stats().TilesTaken() }

func (Conquest) Feat(threshold uint64) string { return "Took " + counted(threshold, "tile", "tiles") }

type Devotion struct{}

func (Devotion) ID() TrackID { return "devotion" }

func (Devotion) Name() string { return "Devotion" }

func (Devotion) Ranks() []Rank { return []Rank{Loyal{}, Devoted{}, Unbroken{}} }

func (Devotion) Progress(career Career) uint64 { return uint64(career.Stats().Streak().Days()) }

func (Devotion) Feat(threshold uint64) string {
	return "Played " + counted(threshold, "day", "days") + " in a row"
}

type Chatter struct{}

func (Chatter) ID() TrackID { return "chatter" }

func (Chatter) Name() string { return "Chatter" }

func (Chatter) Ranks() []Rank { return []Rank{Talker{}, Chatterbox{}, Socialite{}, Icon{}} }

func (Chatter) Progress(career Career) uint64 { return career.Stats().MessagesSent() }

func (Chatter) Feat(threshold uint64) string {
	return "Sent " + counted(threshold, "message", "messages")
}

type Defense struct{}

func (Defense) ID() TrackID { return "defense" }

func (Defense) Name() string { return "Defense" }

func (Defense) Ranks() []Rank { return []Rank{Guard{}, Defender{}, Protector{}, Warden{}} }

func (Defense) Progress(career Career) uint64 { return career.Stats().TilesFortified() }

func (Defense) Feat(threshold uint64) string {
	return "Fortified " + counted(threshold, "tile", "tiles")
}

type Bombs struct{}

func (Bombs) ID() TrackID { return "bombs" }

func (Bombs) Name() string { return "Bombs" }

func (Bombs) Ranks() []Rank { return []Rank{Bomber{}, Blaster{}, Wrecker{}, Destroyer{}} }

func (Bombs) Progress(career Career) uint64 { return career.Stats().BombsDropped() }

func (Bombs) Feat(threshold uint64) string { return "Dropped " + counted(threshold, "bomb", "bombs") }

type Loot struct{}

func (Loot) ID() TrackID { return "loot" }

func (Loot) Name() string { return "Loot" }

func (Loot) Ranks() []Rank { return []Rank{Looter{}, Hunter{}, Pirate{}, Captain{}} }

func (Loot) Progress(career Career) uint64 { return career.Stats().BonusesClaimed() }

func (Loot) Feat(threshold uint64) string {
	return "Caught " + counted(threshold, "bonus box", "bonus boxes")
}

type Quiz struct{}

func (Quiz) ID() TrackID { return "quiz" }

func (Quiz) Name() string { return "Quiz" }

func (Quiz) Ranks() []Rank { return []Rank{Student{}, Expert{}, Professor{}, Genius{}} }

func (Quiz) Progress(career Career) uint64 { return career.Stats().RightAnswers() }

func (Quiz) Feat(threshold uint64) string {
	return "Answered " + counted(threshold, "quiz", "quizzes") + " right"
}

type Missions struct{}

func (Missions) ID() TrackID { return "missions" }

func (Missions) Name() string { return "Missions" }

func (Missions) Ranks() []Rank { return []Rank{Rookie{}, Agent{}, Operative{}, Spymaster{}} }

func (Missions) Progress(career Career) uint64 { return career.Stats().MissionsDone() }

func (Missions) Feat(threshold uint64) string {
	return "Completed " + counted(threshold, "mission", "missions")
}

var english = message.NewPrinter(language.English)

func counted(n uint64, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return english.Sprintf("%d %s", n, many)
}
