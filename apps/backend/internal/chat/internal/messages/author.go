package messages

import (
	"errors"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type AccountID = cpsession.AccountID

var NoAccount = cpsession.NoAccount

func AccountIDOf(value string) AccountID {
	id, err := uuid.Parse(value)
	if err != nil {
		return cpsession.NoAccount
	}
	return AccountID(id)
}

var ErrNoAccount = errors.New("only an account may post or react")

type Author struct {
	Name   string
	Admin  bool
	Color  int32
	Streak uint32
	Title  Title
}

type Title struct {
	ID   string
	Name string
	Rank Rank
}

type Rank struct {
	TrackID   string
	TrackName string
	Number    uint32
	Count     uint32
}

func (t Title) Empty() bool { return t.ID == "" }

func (r Rank) Empty() bool { return r.Count == 0 }

var ErrAuthorUnavailable = errors.New("the sender could not be identified")
