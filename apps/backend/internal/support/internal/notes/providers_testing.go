//go:build testing

package notes

import "sync"

type SequentialIDs struct {
	mu          sync.Mutex
	next        byte
	attachments byte
}

func (s *SequentialIDs) NewID() (NoteID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return NoteID{15: s.next}, nil
}

func (s *SequentialIDs) NewAttachmentID() (AttachmentID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attachments++
	return AttachmentID{0: 0xa, 15: s.attachments}, nil
}
