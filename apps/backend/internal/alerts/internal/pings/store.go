package pings

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
)

type Store interface {
	AttackPingsOf(ctx context.Context, victim nations.Flag, since time.Time) (History, error)
	SaveAttackPing(ctx context.Context, victim nations.Flag, at time.Time) error
	StartPinged(ctx context.Context, key StartKey) (bool, error)
	SaveStartPing(ctx context.Context, key StartKey, at time.Time) error
	RankingPinged(ctx context.Context, round RoundID) (bool, error)
	SaveRankingPing(ctx context.Context, round RoundID, at time.Time) error
}
