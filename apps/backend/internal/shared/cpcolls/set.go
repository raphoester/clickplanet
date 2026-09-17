// Package cpcolls holds the collections the standard library does not.
package cpcolls

import "maps"

// Set is an unordered set of distinct values. Build one with NewSet or
// NewSetWithCapacity: a Set is shared by pointer, like the map it wraps. A nil
// *Set reads as empty, as a nil map does, so a lookup in a map of sets needs no
// ok check.
type Set[T comparable] struct {
	items map[T]struct{}
}

func NewSet[T comparable](items ...T) *Set[T] {
	set := NewSetWithCapacity[T](len(items))
	set.Add(items...)
	return set
}

func NewSetWithCapacity[T comparable](capacity int) *Set[T] {
	return &Set[T]{items: make(map[T]struct{}, capacity)}
}

func (s *Set[T]) Add(items ...T) {
	for _, item := range items {
		s.items[item] = struct{}{}
	}
}

// AddSet adds every item of other.
func (s *Set[T]) AddSet(other *Set[T]) {
	maps.Copy(s.items, other.items)
}

func (s *Set[T]) Delete(items ...T) {
	for _, item := range items {
		delete(s.items, item)
	}
}

func (s *Set[T]) Contains(item T) bool {
	if s == nil {
		return false
	}
	_, ok := s.items[item]
	return ok
}

func (s *Set[T]) Len() int {
	if s == nil {
		return 0
	}
	return len(s.items)
}

func (s *Set[T]) Empty() bool {
	return s.Len() == 0
}

// Clear deletes every item and keeps the memory for the next ones.
func (s *Set[T]) Clear() {
	clear(s.items)
}

// ForEach calls do with every item, in no order. do must not add to or delete from the set.
func (s *Set[T]) ForEach(do func(item T)) {
	if s == nil {
		return
	}
	for item := range s.items {
		do(item)
	}
}
