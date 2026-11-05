//go:build testing

package videos

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

func (s *SequentialIDs) NewID() (VideoID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	return VideoID{s.next}, nil
}

// A bucket in memory: Put stands for the page's PUT on the upload link.
type MemoryBucket struct {
	mu      sync.Mutex
	held    map[VideoID]File
	Deleted []VideoID
	err     error
}

func NewMemoryBucket() *MemoryBucket {
	return &MemoryBucket{held: map[VideoID]File{}}
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
	return fmt.Sprintf("https://bucket.example/clips/videos/%s?size=%d", file.ID, file.Size), nil
}

func (b *MemoryBucket) Held(_ context.Context, id VideoID) (File, error) {
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

func (b *MemoryBucket) Delete(_ context.Context, id VideoID) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.err != nil {
		return b.err
	}
	delete(b.held, id)
	b.Deleted = append(b.Deleted, id)
	return nil
}

type MemoryStore struct {
	mu   sync.Mutex
	kept map[VideoID]VideoSnapshot
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{kept: map[VideoID]VideoSnapshot{}}
}

func (s *MemoryStore) Keep(_ context.Context, video Video) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := video.TakeSnapshot()
	s.kept[snapshot.File.ID] = snapshot
	return nil
}

func (s *MemoryStore) Remove(_ context.Context, id VideoID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.kept[id]; !ok {
		return fmt.Errorf("%w: %s", ErrNoVideo, id)
	}
	delete(s.kept, id)
	return nil
}

func (s *MemoryStore) Kept(id VideoID) (VideoSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept, ok := s.kept[id]
	return kept, ok
}

var ErrBucketDown = errors.New("the bucket is down")
