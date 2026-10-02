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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Persistence interface {
	Load(ctx context.Context, visit func(Stored)) (Marks, error)
	Save(ctx context.Context, changes Changes) error
}

type Stored struct {
	Position ledger.Position
	Taking   ledger.Taking
}

type Marks struct {
	Head      ledger.Position
	Forgotten map[ledger.Caller]ledger.Position
}

type Changes struct {
	From  ledger.Position
	Takes iter.Seq[Stored]
	Marks Marks
}

const flushTimeout = 10 * time.Second

var errCorruptState = errors.New("corrupt stored ledger")

// Never start empty on a failed load: the next flush would delete every stored take.
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

func (s *Storage) Flush(ctx context.Context) error {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	s.mu.Lock()
	head := s.headPositionLocked()
	end := s.next
	from := max(s.saved, head)
	if from == end && head == s.savedHead && s.dirtyMarks.Empty() {
		s.mu.Unlock()
		return nil
	}

	views := s.viewsLocked(from)
	dirty := s.dirtyMarks
	s.dirtyMarks = cpcolls.NewSet[ledger.Caller]()
	forgotten := make(map[ledger.Caller]ledger.Position, dirty.Len())
	dirty.ForEach(func(caller ledger.Caller) {
		if before, ok := s.forgotten[caller]; ok {
			forgotten[caller] = before
		}
	})
	s.mu.Unlock()

	err := s.persistence.Save(ctx, Changes{
		From:  from,
		Takes: storedOf(views),
		Marks: Marks{Head: head, Forgotten: forgotten},
	})

	s.mu.Lock()
	if err != nil {
		s.dirtyMarks.AddSet(dirty)
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
