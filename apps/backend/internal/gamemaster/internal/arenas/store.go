// Package arenas keeps the battles fought on a blank map: each battle's mode, judged once as it starts, and each arena
// planet told closed. The rules of an arena battle's moments are the plan's (moments.Plan).
package arenas

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

type Store interface {
	// KeepJudged is true for a battle not judged yet, and keeps it; one judged already keeps its mode.
	KeepJudged(ctx context.Context, judgement moments.Judgement) (bool, error)
	// KeepClosed is true for an arena not kept closed yet.
	KeepClosed(ctx context.Context, arena moments.RoundID) (bool, error)
	// ArenasKept is every battle judged and every arena closed, in one read.
	ArenasKept(ctx context.Context) (moments.ArenasKept, error)
}
