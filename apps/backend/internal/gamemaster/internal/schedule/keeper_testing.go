//go:build testing

package schedule

import (
	"context"
	"fmt"
	"slices"
)

// FakeKeeper is a module that keeps the sets it is built with.
type FakeKeeper struct {
	Module string
	Sets   []SetName
	Err    error
}

func (f FakeKeeper) Keeps(_ context.Context, set SetName) error {
	if f.Err != nil {
		return f.Err
	}
	if !slices.Contains(f.Sets, set) {
		return fmt.Errorf("%s %w: %q", f.Module, ErrUnknownSet, set)
	}
	return nil
}

// FakeIDs names each entry with the next of its ids.
type FakeIDs struct {
	IDs []EntryID
}

func (f *FakeIDs) NewID() (EntryID, error) {
	if len(f.IDs) == 0 {
		return EntryID{}, fmt.Errorf("no id left")
	}
	id := f.IDs[0]
	f.IDs = f.IDs[1:]
	return id, nil
}
