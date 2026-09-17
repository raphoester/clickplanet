package players

import "context"

// Persistence is where profiles and stats are kept between boots. It is never read after Load.
type Persistence interface {
	Load(ctx context.Context) (Snapshot, error)
	// Save writes the changes in one transaction.
	Save(ctx context.Context, changes Changes) error
}

// Snapshot is every profile and every stats kept.
type Snapshot struct {
	Profiles []Profile
	Stats    []Stats
}

// Changes is one flush: the rows as they are now, and the rows that are gone. An id in both lists of one kind
// never happens.
type Changes struct {
	Profiles []Profile
	Stats    []Stats

	DeletedProfiles []AccountID
	DeletedStats    []AccountID
}

// Empty is a flush with nothing to write.
func (c Changes) Empty() bool {
	return len(c.Profiles) == 0 && len(c.Stats) == 0 && len(c.DeletedProfiles) == 0 && len(c.DeletedStats) == 0
}
