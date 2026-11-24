package submissions

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
)

var (
	ErrWaiting      = errors.New("one of the account's submissions waits for its review already")
	ErrNotFounder   = errors.New("only the account that founded a flag submits its new name or picture")
	ErrDecided      = errors.New("the submission is decided already")
	ErrNotApproved  = errors.New("only an approved submission of a new flag opens one")
	ErrInvalidState = errors.New("a submission kept in no state it can be in")
)

// A rejected picture is deleted this long after the rejection. It never became public.
const rejectedKept = 30 * 24 * time.Hour

type Kind string

const (
	// A flag the game does not have yet.
	New Kind = "new"
	// A new name or picture for a flag opened from the account's own submission.
	Change Kind = "change"
)

type State string

const (
	Waiting  State = "waiting"
	Approved State = "approved"
	Rejected State = "rejected"
	// A flag was opened from it.
	Opened State = "opened"
)

// Draft is what a player typed: the place's name, its pin, and for a change the flag it is for.
type Draft struct {
	Name catalog.Name
	Pin  catalog.Pin
	// Empty for a new flag.
	Flag catalog.FlagID
}

func ParseDraft(name string, latitude, longitude float64, flag string) (Draft, error) {
	parsed, err := catalog.ParseName(name)
	if err != nil {
		return Draft{}, err //nolint:wrapcheck // the parse names it.
	}
	pin, err := catalog.ParsePin(latitude, longitude)
	if err != nil {
		return Draft{}, err //nolint:wrapcheck // the parse names it.
	}
	draft := Draft{Name: parsed, Pin: pin}
	if flag == "" {
		return draft, nil
	}
	draft.Flag, err = catalog.ParseFlagID(flag)
	if err != nil {
		return Draft{}, err //nolint:wrapcheck // the parse names it.
	}
	return draft, nil
}

// Submission is a flag a player submitted for the operator's review, from its submission to its verdict, and for a
// new flag to its opening.
type Submission struct {
	id      SubmissionID
	account AccountID
	kind    Kind
	flag    catalog.FlagID
	name    catalog.Name
	pin     catalog.Pin
	picture pictures.Picture
	others  pictures.OtherColors
	held    bool
	state   State
	// Only a rejected one.
	rejection   Rejection
	submittedAt time.Time
	decidedAt   time.Time
}

// Submit is the account's new submission, refused while another of its submissions waits, and for a change of a flag
// no submission of the account opened.
func Submit(
	id SubmissionID, account AccountID, draft Draft, drawn pictures.Drawn, earlier []Submission, at time.Time,
) (Submission, error) {
	kind := New
	if draft.Flag != "" {
		kind = Change
	}
	founded := false
	for _, submission := range earlier {
		if submission.state == Waiting {
			return Submission{}, fmt.Errorf("%w: %s", ErrWaiting, submission.id)
		}
		founded = founded || (submission.kind == New && submission.state == Opened && submission.flag == draft.Flag)
	}
	if kind == Change && !founded {
		return Submission{}, fmt.Errorf("%w: %s", ErrNotFounder, draft.Flag)
	}
	return Snapshot{
		ID: id, Account: account, Kind: kind, Flag: draft.Flag, Name: draft.Name, Pin: draft.Pin,
		Picture: drawn.Picture.TakeSnapshot(), OtherColors: drawn.OtherColors, Held: true, State: Waiting,
		SubmittedAt: at, DecidedAt: at,
	}.Restore()
}

// Approved makes the picture public: the bucket no longer holds it in private.
func (s Submission) Approved(at time.Time) (Submission, error) {
	if s.state != Waiting {
		return s, fmt.Errorf("%s is %s: %w", s.id, s.state, ErrDecided)
	}
	s.state, s.decidedAt, s.held = Approved, at, false
	return s, nil
}

func (s Submission) Rejected(rejection Rejection, at time.Time) (Submission, error) {
	if s.state != Waiting {
		return s, fmt.Errorf("%s is %s: %w", s.id, s.state, ErrDecided)
	}
	s.state, s.decidedAt, s.rejection = Rejected, at, rejection
	return s, nil
}

// Restyle puts an approved change on its flag: the flag wears the name and the picture approved.
func (s Submission) Restyle(flag catalog.Flag, at time.Time) (catalog.Flag, proto.Message, error) {
	if s.kind != Change || s.state != Approved {
		return flag, nil, fmt.Errorf("%s is a %s submission %s: %w", s.id, s.kind, s.state, ErrNotApproved)
	}
	restyled, fact := flag.Restyled(s.name, s.picture, at)
	return restyled, fact, nil
}

// Open opens the flag of an approved new submission, at its pin, with its name and picture.
func (s Submission) Open(opening catalog.Opening, endsAt time.Time) (Submission, catalog.Flag, error) {
	if s.kind != New || s.state != Approved {
		return s, catalog.Flag{}, fmt.Errorf("%s is a %s submission %s: %w", s.id, s.kind, s.state, ErrNotApproved)
	}
	flag, err := opening.Open(s.name, s.picture, s.pin, endsAt)
	if err != nil {
		return s, catalog.Flag{}, err //nolint:wrapcheck // the opening names it.
	}
	opened := flag.TakeSnapshot()
	s.state, s.flag, s.decidedAt = Opened, opened.ID, opened.DecidedAt
	return s, flag, nil
}

// PictureDeleted is the submission once its rejected picture is deleted, 30 days after the rejection.
func (s Submission) PictureDeleted(now time.Time) (Submission, bool) {
	if s.state != Rejected || !s.held || now.Sub(s.decidedAt) < rejectedKept {
		return s, false
	}
	s.held = false
	return s, true
}

func (s Submission) TakeSnapshot() Snapshot {
	return Snapshot{
		ID: s.id, Account: s.account, Kind: s.kind, Flag: s.flag, Name: s.name, Pin: s.pin,
		Picture: s.picture.TakeSnapshot(), OtherColors: s.others, Held: s.held, State: s.state, Rejection: s.rejection,
		SubmittedAt: s.submittedAt, DecidedAt: s.decidedAt,
	}
}

type Snapshot struct {
	ID      SubmissionID
	Account AccountID
	Kind    Kind
	// The flag a change is for, or the flag opened from a new one; empty until then.
	Flag        catalog.FlagID
	Name        catalog.Name
	Pin         catalog.Pin
	Picture     pictures.Snapshot
	OtherColors pictures.OtherColors
	// The bucket holds the picture in private: until its approval, or 30 days after its rejection.
	Held  bool
	State State
	// Only for a rejected one.
	Rejection   Rejection
	SubmittedAt time.Time
	// When it took its state.
	DecidedAt time.Time
}

func (s Snapshot) Restore() (Submission, error) {
	if uuid.UUID(s.ID) == uuid.Nil || uuid.UUID(s.Account) == uuid.Nil {
		return Submission{}, fmt.Errorf("%w: no id or no account", ErrInvalidState)
	}
	picture, err := s.Picture.Restore()
	if err != nil {
		return Submission{}, fmt.Errorf("the submission %s: %w", s.ID, err)
	}
	if _, err := catalog.ParseName(s.Name.String()); err != nil {
		return Submission{}, fmt.Errorf("the submission %s: %w", s.ID, err)
	}
	if _, err := catalog.ParsePin(s.Pin.Latitude, s.Pin.Longitude); err != nil {
		return Submission{}, fmt.Errorf("the submission %s: %w", s.ID, err)
	}
	if err := s.checkState(); err != nil {
		return Submission{}, fmt.Errorf("the submission %s: %w", s.ID, err)
	}
	return Submission{
		id: s.ID, account: s.Account, kind: s.Kind, flag: s.Flag, name: s.Name, pin: s.Pin, picture: picture,
		others: s.OtherColors, held: s.Held, state: s.State, rejection: s.Rejection, submittedAt: s.SubmittedAt,
		decidedAt: s.DecidedAt,
	}, nil
}

func (s Snapshot) checkState() error {
	switch s.Kind {
	case New, Change:
	default:
		return fmt.Errorf("%w: the kind %q", ErrInvalidState, s.Kind)
	}
	if s.Kind == Change && s.Flag == "" {
		return fmt.Errorf("%w: a change for no flag", ErrInvalidState)
	}
	switch s.State {
	case Waiting, Approved:
	case Rejected:
		if err := s.Rejection.check(); err != nil {
			return err
		}
	case Opened:
		if s.Kind != New || s.Flag == "" {
			return fmt.Errorf("%w: opened, a %s submission of the flag %q", ErrInvalidState, s.Kind, s.Flag)
		}
	default:
		return fmt.Errorf("%w: the state %q", ErrInvalidState, s.State)
	}
	if s.State != Rejected && s.Rejection != (Rejection{}) {
		return fmt.Errorf("%w: a rejection on a %s submission", ErrInvalidState, s.State)
	}
	if s.Held && s.State != Waiting && s.State != Rejected {
		return fmt.Errorf("%w: a picture held private once %s", ErrInvalidState, s.State)
	}
	return nil
}
