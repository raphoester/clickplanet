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
	name   string
	admin  bool
	color  int32
	streak uint32
	title  Title
}

func AuthorOf(name string, admin bool, color int32, streak uint32, title Title) Author {
	return Author{name: name, admin: admin, color: color, streak: streak, title: title}
}

func (a Author) Name() string { return a.name }

func (a Author) Admin() bool { return a.admin }

func (a Author) Color() int32 { return a.color }

func (a Author) Streak() uint32 { return a.streak }

func (a Author) Title() Title { return a.title }

type Title struct {
	id   string
	name string
	rank Rank
}

func TitleOf(id string, name string, rank Rank) Title {
	return Title{id: id, name: name, rank: rank}
}

func (t Title) ID() string { return t.id }

func (t Title) Name() string { return t.name }

func (t Title) Rank() Rank { return t.rank }

func (t Title) Empty() bool { return t.id == "" }

type Rank struct {
	trackID   string
	trackName string
	number    uint32
	count     uint32
}

func RankOf(trackID string, trackName string, number uint32, count uint32) Rank {
	return Rank{trackID: trackID, trackName: trackName, number: number, count: count}
}

func (r Rank) TrackID() string { return r.trackID }

func (r Rank) TrackName() string { return r.trackName }

func (r Rank) Number() uint32 { return r.number }

func (r Rank) Count() uint32 { return r.count }

func (r Rank) Empty() bool { return r.count == 0 }

var ErrAuthorUnavailable = errors.New("the sender could not be identified")
