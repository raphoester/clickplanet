package arrivals

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type ArrivalID uuid.UUID

type AccountID = accounts.AccountID

type Source string

const (
	// A player's recruit link, which is pressed: the link, Invite, the picture.
	SourceRecruit Source = "recruit"
	// The address a player's video shows, which a viewer types.
	SourceRecruitVideo Source = "recruit_video"
	SourceYouTube      Source = "youtube"
	SourceTikTok       Source = "tiktok"
	SourceUnknown      Source = "unknown"
)

func (s Source) known() bool {
	switch s {
	case SourceRecruit, SourceRecruitVideo, SourceYouTube, SourceTikTok, SourceUnknown:
		return true
	default:
		return false
	}
}

// A tracked address's first segment, one letter, since the ones that matter are typed by hand off a video:
// clickplanet.lol/t. The frontend's home page sends it as the tag.
const (
	tiktokTag       = "t"
	youtubeTag      = "y"
	recruitTag      = "r"
	recruitVideoTag = "v"
)

// The tags of the links out before the one-letter addresses (Nov 20 2026): every clip's description says ?s=yt.
const (
	oldRecruitTag = "rc"
	oldYouTubeTag = "yt"
)

// What the browser's first page said, as the client sent it: its ?s=, the host it came from, and the account that held
// the name in the player's address it came by.
type Landing struct {
	Tag       string
	Referrer  string
	Recruiter AccountID
}

const (
	maxHostLength = 253
	maxTagLength  = 32
)

var ErrInvalidArrival = errors.New("the arrival is invalid")

type Arrival struct {
	id       ArrivalID
	account  AccountID
	source   Source
	tag      string
	referrer string
	at       time.Time
}

func NewArrival(id ArrivalID, account AccountID, landing Landing, at time.Time) (Arrival, error) {
	if account == (AccountID{}) {
		return Arrival{}, fmt.Errorf("%w: no account", ErrInvalidArrival)
	}
	tag := tagOf(landing.Tag)
	return Snapshot{
		ID: id, Account: account, Source: sourceOf(tag), Tag: tag, Referrer: hostOf(landing.Referrer), At: at,
	}.Restore()
}

// A tag nobody posted is no source: anybody can type one.
func sourceOf(tag string) Source {
	switch tag {
	case recruitTag, oldRecruitTag:
		return SourceRecruit
	case recruitVideoTag:
		return SourceRecruitVideo
	case youtubeTag, oldYouTubeTag:
		return SourceYouTube
	case tiktokTag:
		return SourceTikTok
	default:
		return SourceUnknown
	}
}

// A short plain word, as a streamer's name: anything longer or with other signs is "".
func tagOf(tag string) string {
	kept := strings.ToLower(strings.TrimSpace(tag))
	if len(kept) > maxTagLength {
		return ""
	}
	for _, r := range kept {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return ""
		}
	}
	return kept
}

// A host name and nothing else: a path, a query, a port or a whole address is "".
func hostOf(referrer string) string {
	host := strings.ToLower(strings.TrimSpace(referrer))
	if len(host) > maxHostLength {
		return ""
	}
	for _, r := range host {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '.' && r != '-' {
			return ""
		}
	}
	return host
}

// A recruit's arrival, by its link or its video, keeps who recruited it; another arrival has no recruiter, whatever its
// landing said.
func (a Arrival) Recruitment(recruiter AccountID) (Recruitment, bool) {
	if (a.source != SourceRecruit && a.source != SourceRecruitVideo) || recruiter == (AccountID{}) {
		return Recruitment{}, false
	}
	return Recruitment{arrival: a.id, recruiter: recruiter}, true
}

// The arrival with nothing that says whose it was: the source, the tag, the referrer and the time stay.
func (a Arrival) Anonymized() Arrival {
	a.account = AccountID{}
	return a
}

func AllAnonymized(kept []Arrival) []Arrival {
	anonymized := make([]Arrival, 0, len(kept))
	for _, arrival := range kept {
		anonymized = append(anonymized, arrival.Anonymized())
	}
	return anonymized
}

func (a Arrival) TakeSnapshot() Snapshot {
	return Snapshot{ID: a.id, Account: a.account, Source: a.source, Tag: a.tag, Referrer: a.referrer, At: a.at}
}

type Snapshot struct {
	ID ArrivalID
	// Zero once the arrival is anonymized.
	Account AccountID
	Source  Source
	Tag     string
	// The host the browser came from, "" for none.
	Referrer string
	At       time.Time
}

func (s Snapshot) Restore() (Arrival, error) {
	switch {
	case s.ID == ArrivalID{}:
		return Arrival{}, fmt.Errorf("%w: no id", ErrInvalidArrival)
	case !s.Source.known():
		return Arrival{}, fmt.Errorf("%w: unknown source %q", ErrInvalidArrival, string(s.Source))
	case s.At.IsZero():
		return Arrival{}, fmt.Errorf("%w: no time", ErrInvalidArrival)
	}
	return Arrival{id: s.ID, account: s.Account, source: s.Source, tag: s.Tag, referrer: s.Referrer, at: s.At}, nil
}

// A store finds arrivals and saves them. It holds no rule: what changes an arrival is a method of Arrival.
type Store interface {
	// An arrival saved again replaces itself. All of them, or none.
	Save(ctx context.Context, arrivals ...Arrival) error
	// The arrivals that still name the account, oldest first.
	Of(ctx context.Context, account AccountID) ([]Arrival, error)
}

type IDProvider interface {
	NewID() (ArrivalID, error)
}
