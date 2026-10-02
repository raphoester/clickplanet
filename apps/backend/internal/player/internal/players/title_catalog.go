package players

func NewCatalog() Catalog {
	return Catalog{Settler{}, Governor{}, Conqueror{}, Emperor{}, Loyal{}, Devoted{}, Unbroken{}}
}

type Settler struct{}

func (Settler) ID() TitleID { return "settler" }

func (Settler) Name() string { return "Settler" }

func (Settler) EarnedBy(stats Stats) bool { return stats.TilesTaken >= 100 }

type Governor struct{}

func (Governor) ID() TitleID { return "governor" }

func (Governor) Name() string { return "Governor" }

func (Governor) EarnedBy(stats Stats) bool { return stats.TilesTaken >= 1_000 }

type Conqueror struct{}

func (Conqueror) ID() TitleID { return "conqueror" }

func (Conqueror) Name() string { return "Conqueror" }

func (Conqueror) EarnedBy(stats Stats) bool { return stats.TilesTaken >= 10_000 }

type Emperor struct{}

func (Emperor) ID() TitleID { return "emperor" }

func (Emperor) Name() string { return "Emperor" }

func (Emperor) EarnedBy(stats Stats) bool { return stats.TilesTaken >= 100_000 }

type Loyal struct{}

func (Loyal) ID() TitleID { return "loyal" }

func (Loyal) Name() string { return "Loyal" }

func (Loyal) EarnedBy(stats Stats) bool { return stats.StreakBest >= 7 }

type Devoted struct{}

func (Devoted) ID() TitleID { return "devoted" }

func (Devoted) Name() string { return "Devoted" }

func (Devoted) EarnedBy(stats Stats) bool { return stats.StreakBest >= 30 }

type Unbroken struct{}

func (Unbroken) ID() TitleID { return "unbroken" }

func (Unbroken) Name() string { return "Unbroken" }

func (Unbroken) EarnedBy(stats Stats) bool { return stats.StreakBest >= 100 }
