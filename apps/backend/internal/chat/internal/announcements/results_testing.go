//go:build testing

package announcements

import (
	"context"
	"errors"
)

var ErrNotScored = errors.New("scoring has no such id")

type FakeResults struct {
	Rounds map[RoundID][]Place
	Tables map[BookID][]Place
	Err    error
}

func (f FakeResults) Round(_ context.Context, round RoundID) ([]Place, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	places, ok := f.Rounds[round]
	if !ok {
		return nil, ErrNotScored
	}
	return places, nil
}

func (f FakeResults) Table(_ context.Context, book BookID) ([]Place, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	places, ok := f.Tables[book]
	if !ok {
		return nil, ErrNotScored
	}
	return places, nil
}
