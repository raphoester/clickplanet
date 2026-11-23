package catalog

import "context"

type Store interface {
	// ErrUnknownFlag for a flag never opened.
	Flag(ctx context.Context, id FlagID) (Flag, error)
	// Every flag opened since the game shipped, the closed ones too.
	Flags(ctx context.Context) ([]Flag, error)
	Save(ctx context.Context, flags ...Flag) error
}
