//go:build testing

package backgrounds

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type SequentialIDs struct {
	mu   sync.Mutex
	next byte
}

func (s *SequentialIDs) NewID() (BackgroundID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	return BackgroundID{15: s.next}, nil
}

// A bucket in memory: Put stands for the page's PUT on the upload link.
type MemoryBucket struct {
	mu      sync.Mutex
	held    map[BackgroundID]File
	Deleted []BackgroundID
	err     error
}

func NewMemoryBucket() *MemoryBucket {
	return &MemoryBucket{held: map[BackgroundID]File{}}
}

func (b *MemoryBucket) Failing(err error) *MemoryBucket {
	b.err = err
	return b
}

func (b *MemoryBucket) Put(file File) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.held[file.ID] = file
}

func (b *MemoryBucket) UploadURL(file File) (string, error) {
	if b.err != nil {
		return "", b.err
	}
	return fmt.Sprintf("https://bucket.example/promo/backgrounds/%s?size=%d", file.ID, file.Size), nil
}

func (b *MemoryBucket) PublicURL(id BackgroundID) string {
	return "https://media.example/promo/backgrounds/" + id.String()
}

func (b *MemoryBucket) Held(_ context.Context, id BackgroundID) (File, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.err != nil {
		return File{}, b.err
	}
	file, ok := b.held[id]
	if !ok {
		return File{}, fmt.Errorf("%w: %s", ErrNotUploaded, id)
	}
	return file, nil
}

func (b *MemoryBucket) Delete(_ context.Context, id BackgroundID) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.err != nil {
		return b.err
	}
	delete(b.held, id)
	b.Deleted = append(b.Deleted, id)
	return nil
}

func (b *MemoryBucket) Holds(id BackgroundID) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, ok := b.held[id]
	return ok
}

// The library in memory. Shown names the sets whose cards show each background, as the cards would.
type MemoryStore struct {
	mu    sync.Mutex
	kept  map[BackgroundID]BackgroundSnapshot
	Shown map[BackgroundID][]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{kept: map[BackgroundID]BackgroundSnapshot{}, Shown: map[BackgroundID][]string{}}
}

func (s *MemoryStore) Keep(_ context.Context, background Background) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := background.TakeSnapshot()
	s.kept[snapshot.File.ID] = snapshot
	return nil
}

func (s *MemoryStore) Remove(_ context.Context, id BackgroundID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sets := s.Shown[id]; len(sets) > 0 {
		return fmt.Errorf("%w: %s, by %v", ErrShown, id, sets)
	}
	if _, ok := s.kept[id]; !ok {
		return fmt.Errorf("%w: %s", ErrNoBackground, id)
	}
	delete(s.kept, id)
	return nil
}

func (s *MemoryStore) Kept(id BackgroundID) (BackgroundSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept, ok := s.kept[id]
	return kept, ok
}

var ErrBucketDown = errors.New("the bucket is down")
