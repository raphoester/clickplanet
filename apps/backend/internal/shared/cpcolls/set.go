package cpcolls

import "maps"

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

func (s *Set[T]) Clear() {
	clear(s.items)
}

// ForEach visits in no order, and do must not add to or delete from the set.
func (s *Set[T]) ForEach(do func(item T)) {
	if s == nil {
		return
	}
	for item := range s.items {
		do(item)
	}
}
