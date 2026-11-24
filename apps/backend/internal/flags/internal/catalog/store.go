package catalog

import (
	"context"
	"time"
)

type Store interface {
	// ErrUnknownFlag for a flag never opened.
	Flag(ctx context.Context, id FlagID) (Flag, error)
	// Every flag opened since the game shipped, the closed ones too.
	Flags(ctx context.Context) ([]Flag, error)
	// Ending is every flag open whose end is at or before the time.
	Ending(ctx context.Context, at time.Time) ([]Flag, error)
	// ErrNameTaken when another flag has its name, whatever its case.
	Save(ctx context.Context, flags ...Flag) error
}
