//go:build testing

package inmemory_upload_storage

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/uploads"
)

func New() *Storage {
	return &Storage{sent: cpcolls.NewSet[notes.AttachmentID]()}
}

type Storage struct {
	mu   sync.Mutex
	kept []uploads.Snapshot
	sent *cpcolls.Set[notes.AttachmentID]
	err  error
}

var _ uploads.Storage = (*Storage)(nil)

func (s *Storage) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.err = err
}

// The files a note names, as the note store would hold them.
func (s *Storage) Send(ids ...notes.AttachmentID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sent.Add(ids...)
}

func (s *Storage) Kept() []uploads.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.kept)
}

func (s *Storage) Held(context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return 0, s.err
	}
	return s.held(), nil
}

func (s *Storage) held() int64 {
	var bytes int64
	for _, kept := range s.kept {
		bytes += kept.Size
	}
	return bytes
}

func (s *Storage) Reserve(_ context.Context, upload uploads.Upload, check func(held int64) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}
	if err := check(s.held()); err != nil {
		return err
	}
	s.write(upload.TakeSnapshot())
	return nil
}

func (s *Storage) Unsent(_ context.Context, cutoff time.Time, limit int) ([]uploads.Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}

	ordered := slices.Clone(s.kept)
	slices.SortStableFunc(ordered, func(a, b uploads.Snapshot) int { return a.At.Compare(b.At) })

	var unsent []uploads.Upload
	for _, kept := range ordered {
		if len(unsent) == limit {
			break
		}
		if !kept.At.Before(cutoff) || s.sent.Contains(kept.ID) {
			continue
		}
		upload, err := kept.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore the upload %s: %w", kept.ID, err)
		}
		unsent = append(unsent, upload)
	}
	return unsent, nil
}

func (s *Storage) Save(_ context.Context, saved ...uploads.Upload) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}
	for _, upload := range saved {
		s.write(upload.TakeSnapshot())
	}
	return nil
}

func (s *Storage) write(kept uploads.Snapshot) {
	s.kept = slices.DeleteFunc(s.kept, func(old uploads.Snapshot) bool { return old.ID == kept.ID })
	if !kept.Discarded {
		s.kept = append(s.kept, kept)
	}
}
