package titles

import "time"

func NewCatalog() Catalog {
	return Catalog{OG{}, Settler{}, Governor{}, Conqueror{}, Emperor{}, Loyal{}, Devoted{}, Unbroken{}}
}

var ogCutoff = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

type OG struct{}

func (OG) ID() ID { return "og" }

func (OG) Name() string { return "OG" }

func (OG) EarnedBy(career Career) bool {
	return !career.CreatedAt.IsZero() && career.CreatedAt.Before(ogCutoff)
}

type Settler struct{}

func (Settler) ID() ID { return "settler" }

func (Settler) Name() string { return "Settler" }

func (Settler) EarnedBy(career Career) bool { return career.Stats.TilesTaken >= 100 }

type Governor struct{}

func (Governor) ID() ID { return "governor" }

func (Governor) Name() string { return "Governor" }

func (Governor) EarnedBy(career Career) bool { return career.Stats.TilesTaken >= 1_000 }

type Conqueror struct{}

func (Conqueror) ID() ID { return "conqueror" }

func (Conqueror) Name() string { return "Conqueror" }

func (Conqueror) EarnedBy(career Career) bool { return career.Stats.TilesTaken >= 10_000 }

type Emperor struct{}

func (Emperor) ID() ID { return "emperor" }

func (Emperor) Name() string { return "Emperor" }

func (Emperor) EarnedBy(career Career) bool { return career.Stats.TilesTaken >= 100_000 }

type Loyal struct{}

func (Loyal) ID() ID { return "loyal" }

func (Loyal) Name() string { return "Loyal" }

func (Loyal) EarnedBy(career Career) bool { return career.Stats.StreakBest >= 7 }

type Devoted struct{}

func (Devoted) ID() ID { return "devoted" }

func (Devoted) Name() string { return "Devoted" }

func (Devoted) EarnedBy(career Career) bool { return career.Stats.StreakBest >= 30 }

type Unbroken struct{}

func (Unbroken) ID() ID { return "unbroken" }

func (Unbroken) Name() string { return "Unbroken" }

func (Unbroken) EarnedBy(career Career) bool { return career.Stats.StreakBest >= 100 }
