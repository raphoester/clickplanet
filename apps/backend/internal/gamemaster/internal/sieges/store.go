// Package sieges keeps the battles played on a blank map: each battle's mode, judged once as it starts, and each arena
// planet told closed. The rules of a siege's moments are the plan's (moments.Plan).
package sieges

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

type Store interface {
	// KeepJudged is true for a battle not judged yet, and keeps it; one judged already keeps its mode.
	KeepJudged(ctx context.Context, judgement moments.Judgement) (bool, error)
	Judged(ctx context.Context) ([]moments.Judgement, error)
	// KeepClosed is true for an arena not kept closed yet.
	KeepClosed(ctx context.Context, arena moments.RoundID) (bool, error)
	Closed(ctx context.Context) ([]moments.RoundID, error)
}
