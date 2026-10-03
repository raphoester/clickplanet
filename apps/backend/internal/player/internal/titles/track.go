package titles

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
}

type Conquest struct{}

func (Conquest) ID() TrackID { return "conquest" }

func (Conquest) Name() string { return "Conquest" }

func (Conquest) Ranks() []Rank {
	return []Rank{Settler{}, Raider{}, Warlord{}, Conqueror{}, Warmaster{}}
}

func (Conquest) Progress(career Career) uint64 { return career.Stats.TilesTaken }

type Devotion struct{}

func (Devotion) ID() TrackID { return "devotion" }

func (Devotion) Name() string { return "Devotion" }

func (Devotion) Ranks() []Rank { return []Rank{Loyal{}, Devoted{}, Unbroken{}} }

func (Devotion) Progress(career Career) uint64 { return uint64(career.Stats.StreakCurrent) }
