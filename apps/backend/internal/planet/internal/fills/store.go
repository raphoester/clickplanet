package fills

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

type Store interface {
	// Fill is the fill kept for the request, or none.
	Fill(ctx context.Context, request bonuses.FillID) (Fill, error)
	// Unfinished is the fill started first of those still running, or none.
	Unfinished(ctx context.Context) (Fill, error)
	Save(ctx context.Context, fill Fill) error
}
