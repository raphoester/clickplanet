package inmemory_ledger_storage

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

// Persistence is where the ledger is kept between boots. It is never read after Load.
type Persistence interface {
	// Load visits every stored take in position order, and returns the marks.
	Load(ctx context.Context, visit func(Stored)) (Marks, error)
	// Save writes the changes in one transaction.
	Save(ctx context.Context, changes Changes) error
}

// Stored is a take with its place in the ledger.
type Stored struct {
	Position ledger.Position
	Taking   ledger.Taking
}

// Marks bound a replay: every take before Head is gone, and a caller's takes before its mark are forgotten.
type Marks struct {
	Head      ledger.Position
	Forgotten map[ledger.Caller]ledger.Position
}

// Changes is one flush. Takes start at From or later, and a stored take at or past From is written again.
type Changes struct {
	From  ledger.Position
	Takes iter.Seq[Stored]
	// Marks holds the current head, and only the forgotten marks set since the last flush.
	Marks Marks
}

const flushTimeout = 10 * time.Second

var errCorruptState = errors.New("corrupt stored ledger")

// Load refuses rather than start empty: an empty ledger that then flushes would lose every take on record.
func (s *Storage) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var restoreErr error
	marks, err := s.persistence.Load(ctx, func(take Stored) {
		if restoreErr != nil {
			return
		}
		restoreErr = s.restoreLocked(take.Position, take.Taking)
	})
	if err != nil {
		return fmt.Errorf("failed to read the stored ledger: %w", err)
	}
	if restoreErr != nil {
		return fmt.Errorf("failed to restore the stored ledger: %w", restoreErr)
	}

	s.applyMarksLocked(marks.Head, maps.Clone(marks.Forgotten))
	s.saved = s.next
	s.savedHead = marks.Head

	s.logger.Info("loaded the ledger", slog.Int("takings", s.liveLocked()))

	return nil
}

// restoreLocked appends a take at its stored position. A gap is takes dropped before they were stored.
func (s *Storage) restoreLocked(position ledger.Position, taking ledger.Taking) error {
	if position < s.next {
		return fmt.Errorf("%w: take at %d overlaps takes up to %d", errCorruptState, position, s.next)
	}

	if position != s.next {
		if n := len(s.chunks); n > 0 {
			s.sealLocked(s.chunks[n-1])
		}
		s.next = position
	}
	s.appendLocked(taking)

	return nil
}

func (s *Storage) applyMarksLocked(head ledger.Position, forgotten map[ledger.Caller]ledger.Position) {
	if s.headPositionLocked() < head {
		s.next = max(s.next, head)
		s.dropLocked(int(head - s.headPositionLocked())) //nolint:gosec // bounded by what was loaded.
	}

	if forgotten == nil {
		forgotten = make(map[ledger.Caller]ledger.Position)
	}
	s.forgotten = forgotten
	s.forgetMarksLocked()
}

func (s *Storage) Name() string { return "ledger-storage" }

func (s *Storage) Run(ctx context.Context) {
	ticker := time.NewTicker(s.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.flushOrLog(ctx)
		case <-ctx.Done():
			s.flushOrLog(context.WithoutCancel(ctx))
			return
		}
	}
}

func (s *Storage) flushOrLog(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, flushTimeout)
	defer cancel()

	if err := s.Flush(ctx); err != nil {
		s.logger.Error("failed to flush the ledger, retrying next tick", slog.Any("error", err))
	}
}

// Flush writes the takes appended since the last flush and the marks that moved. A failed write keeps them for the next one.
func (s *Storage) Flush(ctx context.Context) error {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	s.mu.Lock()
	head := s.headPositionLocked()
	end := s.next
	from := max(s.saved, head)
	if from == end && head == s.savedHead && len(s.dirtyMarks) == 0 {
		s.mu.Unlock()
		return nil
	}

	views := s.viewsLocked(from)
	dirty := s.dirtyMarks
	s.dirtyMarks = make(map[ledger.Caller]struct{})
	forgotten := make(map[ledger.Caller]ledger.Position, len(dirty))
	for caller := range dirty {
		if before, ok := s.forgotten[caller]; ok {
			forgotten[caller] = before
		}
	}
	s.mu.Unlock()

	err := s.persistence.Save(ctx, Changes{
		From:  from,
		Takes: storedOf(views),
		Marks: Marks{Head: head, Forgotten: forgotten},
	})

	s.mu.Lock()
	if err != nil {
		maps.Copy(s.dirtyMarks, dirty)
		s.mu.Unlock()
		return fmt.Errorf("failed to save %d takes: %w", end-from, err)
	}
	s.saved = end
	s.savedHead = head
	s.mu.Unlock()

	return nil
}

func storedOf(views []view) iter.Seq[Stored] {
	return func(yield func(Stored) bool) {
		for _, v := range views {
			for i, r := range v.records {
				if !yield(Stored{Position: v.first + ledger.Position(i), Taking: v.taking(r)}) { //nolint:gosec // i < chunkSize.
					return
				}
			}
		}
	}
}
