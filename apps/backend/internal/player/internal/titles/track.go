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

func (Conquest) Progress(career Career) uint64 { return career.Stats().TilesTaken() }

type Devotion struct{}

func (Devotion) ID() TrackID { return "devotion" }

func (Devotion) Name() string { return "Devotion" }

func (Devotion) Ranks() []Rank { return []Rank{Loyal{}, Devoted{}, Unbroken{}} }

func (Devotion) Progress(career Career) uint64 { return uint64(career.Stats().Streak().Days()) }

type Chatter struct{}

func (Chatter) ID() TrackID { return "chatter" }

func (Chatter) Name() string { return "Chatter" }

func (Chatter) Ranks() []Rank { return []Rank{Talker{}, Chatterbox{}, Socialite{}, Icon{}} }

func (Chatter) Progress(career Career) uint64 { return career.Stats().MessagesSent() }

type Defense struct{}

func (Defense) ID() TrackID { return "defense" }

func (Defense) Name() string { return "Defense" }

func (Defense) Ranks() []Rank { return []Rank{Guard{}, Defender{}, Protector{}, Warden{}} }

func (Defense) Progress(career Career) uint64 { return career.Stats().TilesFortified() }

type Bombs struct{}

func (Bombs) ID() TrackID { return "bombs" }

func (Bombs) Name() string { return "Bombs" }

func (Bombs) Ranks() []Rank { return []Rank{Bomber{}, Blaster{}, Wrecker{}, Destroyer{}} }

func (Bombs) Progress(career Career) uint64 { return career.Stats().BombsDropped() }

type Loot struct{}

func (Loot) ID() TrackID { return "loot" }

func (Loot) Name() string { return "Loot" }

func (Loot) Ranks() []Rank { return []Rank{Looter{}, Hunter{}, Pirate{}, Captain{}} }

func (Loot) Progress(career Career) uint64 { return career.Stats().BonusesClaimed() }

type Quiz struct{}

func (Quiz) ID() TrackID { return "quiz" }

func (Quiz) Name() string { return "Quiz" }

func (Quiz) Ranks() []Rank { return []Rank{Student{}, Expert{}, Professor{}, Genius{}} }

func (Quiz) Progress(career Career) uint64 { return career.Stats().RightAnswers() }

type Missions struct{}

func (Missions) ID() TrackID { return "missions" }

func (Missions) Name() string { return "Missions" }

func (Missions) Ranks() []Rank { return []Rank{Rookie{}, Agent{}, Operative{}, Spymaster{}} }

func (Missions) Progress(career Career) uint64 { return career.Stats().MissionsDone() }
