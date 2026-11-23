package world_replay

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/forgetting"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Forgettings interface {
	Forgetting(ctx context.Context) (forgetting.Forgetting, error)
}

func NewErasures(replay *PostgresReplay, forgettings Forgettings, clock cptime.Clock) *Erasures {
	return &Erasures{replay: replay, forgettings: forgettings, clock: clock}
}

// Erasures keeps the erasure of the world's ledger before its head, read from the last blank map: a rewind from the
// head sees through what a revert forgot before it. It moves on with the head, and is read again after a revert.
type Erasures struct {
	replay      *PostgresReplay
	forgettings Forgettings
	clock       cptime.Clock

	mu   sync.Mutex
	kept *erased
}

// erased has read every event before a position, as the reverts had forgotten them.
type erased struct {
	forgotten forgetting.Forgetting
	before    ledger.Position
	erasing   *erasing
}

// The events between the kept position and the head keep their order by the primary key.
const erasedEvents = `
	SELECT position, map, board, kind, tile, COALESCE(scope, ''), COALESCE(account::text, ''), country, previous,
		taken_at, payload, forgotten
	FROM ledger_events
	WHERE position >= $1 AND position < $2 AND taken_at >= $3
	ORDER BY position
`

// Before answers an erasure of the caller's own, which has read every event before position.
func (e *Erasures) Before(ctx context.Context, position ledger.Position) (*ledger.Erasure, error) {
	forgotten, err := e.forgettings.Forgetting(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read what the reverts forgot: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	next, plan, err := e.from(ctx, forgotten, position)
	if err != nil {
		return nil, err
	}
	resets, cuts := after(next.erasing.at, plan.resets, plan.cuts)
	if _, _, _, err := e.replay.play(ctx, next.erasing, resets, cuts,
		erasedEvents, int64(next.before), int64(position), plan.blank, //nolint:gosec // a position fits a bigint.
	); err != nil {
		return nil, fmt.Errorf("failed to read the ledger before its head: %w", err)
	}
	next.before = position
	e.kept = &next
	return next.erasing.erasure.Clone(), nil
}

// from is a copy of the erasure kept, or a fresh one at the last blank map: after a revert, or for a position behind it.
func (e *Erasures) from(ctx context.Context, forgotten forgetting.Forgetting, position ledger.Position) (erased, Plan, error) {
	now := e.clock.Now()
	if kept := e.kept; kept != nil && kept.forgotten == forgotten && kept.before <= position {
		plan, err := e.replay.Plan(ctx, kept.erasing.at, now)
		if err != nil {
			return erased{}, Plan{}, fmt.Errorf("failed to plan the world's replay: %w", err)
		}
		return erased{forgotten: forgotten, before: kept.before, erasing: kept.erasing.clone()}, plan, nil
	}
	plan, err := e.replay.Plan(ctx, now, now)
	if err != nil {
		return erased{}, Plan{}, fmt.Errorf("failed to plan the world's replay: %w", err)
	}
	return erased{forgotten: forgotten, erasing: &erasing{erasure: ledger.NewErasure(), at: plan.blank}}, plan, nil
}

// after keeps the resets and the cuts after at: the ones at or before it were played before the event read at it.
func after(at time.Time, resets []time.Time, cuts []cut) ([]time.Time, []cut) {
	for len(resets) > 0 && !resets[0].After(at) {
		resets = resets[1:]
	}
	for len(cuts) > 0 && !cuts[0].era.StartedAt.After(at) {
		cuts = cuts[1:]
	}
	return resets, cuts
}

// erasing reads every stage's events into one erasure, which keeps each stage apart. at is the time of the last event
// read, or of the blank map before any.
type erasing struct {
	erasure *ledger.Erasure
	at      time.Time
}

var _ ledger.Replayed = (*erasing)(nil)

func (e *erasing) clone() *erasing {
	return &erasing{erasure: e.erasure.Clone(), at: e.at}
}

func (e *erasing) Done(ledger.Entry) bool { return false }

func (e *erasing) Record(entry ledger.Entry) error {
	if _, _, err := e.erasure.Read(entry); err != nil {
		return err //nolint:wrapcheck // play names the event.
	}
	e.at = entry.At
	return nil
}

func (e *erasing) Wipe(time.Time) { e.erasure.Wipe() }

func (e *erasing) Cut(time.Time, clicks.BoardID, ledger.Carry) { e.erasure.Wipe() }
