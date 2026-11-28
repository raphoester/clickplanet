package submissions

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
)

var (
	ErrWaiting      = errors.New("another of the account's submissions waits for its review")
	ErrNotWaiting   = errors.New("the submission to replace no longer waits: the operator decided it")
	ErrDecided      = errors.New("the submission is decided already")
	ErrNotApproved  = errors.New("only an approved submission opens a flag")
	ErrInvalidState = errors.New("a submission kept in no state it can be in")
	ErrNotYours     = errors.New("only its founder pays for a submission or its flag")
	ErrNotPayable   = errors.New("a rejected submission is not paid for")
	ErrPaidAlready  = errors.New("the submission's time is paid already")
	ErrPaid         = errors.New("a submission paid for is not changed")
	ErrAppealed     = errors.New("a submission a person looks at again is not changed")
	ErrNotRejected  = errors.New("only a rejected submission is looked at again")
	ErrPictureGone  = errors.New("the rejected picture was deleted: nothing is left to look at again")
)

// A rejected picture is deleted this long after the rejection. It never became public.
const rejectedKept = 30 * 24 * time.Hour

type State string

const (
	Waiting  State = "waiting"
	Approved State = "approved"
	Rejected State = "rejected"
	// A flag was opened from it.
	Opened State = "opened"
)

// Draft is what a player typed: the place's name and its pin, the anthem it chose, and the account's submission that
// waits, which it replaces.
type Draft struct {
	Name catalog.Name
	Pin  catalog.Pin
	// The zero Anthem for none.
	Anthem catalog.Anthem
	// The zero id for none.
	Replaces SubmissionID
}

func ParseDraft(name string, latitude, longitude float64, anthem catalog.Anthem, replaces string) (Draft, error) {
	parsed, err := catalog.ParseName(name)
	if err != nil {
		return Draft{}, err //nolint:wrapcheck // the parse names it.
	}
	pin, err := catalog.ParsePin(latitude, longitude)
	if err != nil {
		return Draft{}, err //nolint:wrapcheck // the parse names it.
	}
	draft := Draft{Name: parsed, Pin: pin, Anthem: anthem}
	if replaces == "" {
		return draft, nil
	}
	draft.Replaces, err = ParseSubmissionID(replaces)
	if err != nil {
		return Draft{}, err //nolint:wrapcheck // the parse names it.
	}
	return draft, nil
}

// Submission is a flag a player submitted for the operator's review, from its submission to its verdict, and for a
// new flag to its opening.
type Submission struct {
	id      SubmissionID
	account catalog.AccountID
	flag    catalog.FlagID
	name    catalog.Name
	pin     catalog.Pin
	anthem  catalog.Anthem
	picture pictures.Picture
	others  pictures.OtherColors
	held    bool
	paid    bool
	// Its player asked a person to look again at the judge's rejection.
	appealed bool
	state    State
	// Only a rejected one.
	rejection   Rejection
	submittedAt time.Time
	decidedAt   time.Time
}

// Submit is the account's new submission, refused while another of its submissions waits. It may replace the one
// that waits, never one the operator decided: the ids it replaces are deleted unreviewed, with their pictures.
func Submit(
	id SubmissionID, account catalog.AccountID, draft Draft, drawn pictures.Drawn, earlier []Submission, at time.Time,
) (Submission, []SubmissionID, error) {
	var replaced []SubmissionID
	for _, submission := range earlier {
		switch {
		case submission.id == draft.Replaces && submission.state == Waiting && submission.paid:
			return Submission{}, nil, fmt.Errorf("%w: %s", ErrPaid, submission.id)
		case submission.id == draft.Replaces && submission.state == Waiting && submission.appealed:
			return Submission{}, nil, fmt.Errorf("%w: %s", ErrAppealed, submission.id)
		case submission.id == draft.Replaces && submission.state == Waiting:
			replaced = append(replaced, submission.id)
		case submission.state == Waiting:
			return Submission{}, nil, fmt.Errorf("%w: %s", ErrWaiting, submission.id)
		}
	}
	if draft.Replaces != (SubmissionID{}) && len(replaced) == 0 {
		return Submission{}, nil, fmt.Errorf("%w: %s", ErrNotWaiting, draft.Replaces)
	}
	submission, err := Snapshot{
		ID: id, Account: account, Name: draft.Name, Pin: draft.Pin, Anthem: draft.Anthem.Library(),
		Picture:     drawn.Picture.TakeSnapshot(),
		OtherColors: drawn.OtherColors, Held: true, State: Waiting, SubmittedAt: at, DecidedAt: at,
	}.Restore()
	return submission, replaced, err
}

// Approved makes the picture public: the bucket no longer holds it in private. Two flags never share a name, so a name
// another flag has is no flag to approve.
func (s Submission) Approved(names catalog.Names, at time.Time) (Submission, error) {
	if s.state != Waiting {
		return s, fmt.Errorf("%s is %s: %w", s.id, s.state, ErrDecided)
	}
	if err := names.Free(s.name, ""); err != nil {
		return s, err //nolint:wrapcheck // the names say which.
	}
	s.state, s.decidedAt, s.held = Approved, at, false
	return s, nil
}

// Moved is the submission with its capital where the operator put it, before its verdict.
func (s Submission) Moved(pin catalog.Pin) (Submission, error) {
	if s.state != Waiting {
		return s, fmt.Errorf("%s is %s: %w", s.id, s.state, ErrDecided)
	}
	s.pin = pin
	return s, nil
}

func (s Submission) Rejected(rejection Rejection, at time.Time) (Submission, error) {
	if s.state != Waiting {
		return s, fmt.Errorf("%s is %s: %w", s.id, s.state, ErrDecided)
	}
	s.state, s.decidedAt, s.rejection = Rejected, at, rejection
	return s, nil
}

// Reopened is a rejected submission waiting again for a person, at its founder's appeal, while its picture is held: one
// of the account's submissions waits at a time.
func (s Submission) Reopened(account catalog.AccountID, founded []Submission, at time.Time) (Submission, error) {
	switch {
	case s.account != account:
		return s, fmt.Errorf("%w: %s", ErrNotYours, s.id)
	case s.state != Rejected:
		return s, fmt.Errorf("%s is %s: %w", s.id, s.state, ErrNotRejected)
	case !s.held:
		return s, fmt.Errorf("%w: %s", ErrPictureGone, s.id)
	}
	for _, other := range founded {
		if other.id != s.id && other.state == Waiting {
			return s, fmt.Errorf("%w: %s", ErrWaiting, other.id)
		}
	}
	s.state, s.decidedAt, s.rejection, s.appealed = Waiting, at, Rejection{}, true
	return s, nil
}

// PaidFor is the submission once a payment's time is held for it.
func (s Submission) PaidFor() Submission {
	s.paid = true
	return s
}

// Open opens the flag of an approved submission, at its pin, with its name and picture, held by its founder, unless a
// flag took its name since.
func (s Submission) Open(opening catalog.Opening, names catalog.Names, endsAt time.Time) (Submission, catalog.Flag, error) {
	if s.state != Approved {
		return s, catalog.Flag{}, fmt.Errorf("%s is %s: %w", s.id, s.state, ErrNotApproved)
	}
	if err := names.Free(s.name, ""); err != nil {
		return s, catalog.Flag{}, err //nolint:wrapcheck // the names say which.
	}
	flag, err := opening.OpenFor(s.account, s.name, s.picture, s.pin, s.anthem, endsAt)
	if err != nil {
		return s, catalog.Flag{}, err //nolint:wrapcheck // the opening names it.
	}
	opened := flag.TakeSnapshot()
	s.state, s.flag, s.decidedAt = Opened, opened.ID, opened.DecidedAt
	return s, flag, nil
}

// PayableBy is nil when the account may pay for the submission: its founder, while it waits with no time paid (the
// time is held until its approval), once approved with none (the payment opens its flag), or opened (the time goes to
// its flag, as the flag's rules say).
func (s Submission) PayableBy(account catalog.AccountID) error {
	if s.account != account {
		return fmt.Errorf("%w: %s", ErrNotYours, s.id)
	}
	switch s.state {
	case Waiting, Approved:
		if s.paid {
			return fmt.Errorf("%w: %s", ErrPaidAlready, s.id)
		}
		return nil
	case Opened:
		return nil
	case Rejected:
		return fmt.Errorf("%s is %s: %w", s.id, s.state, ErrNotPayable)
	}
	return fmt.Errorf("%w: the state %q", ErrInvalidState, s.state)
}

// Waiting is true until the operator's verdict: a payment's time is held for it till then.
func (s Submission) Waiting() bool {
	return s.state == Waiting
}

// WaitsPaidFor is true while it waits with its time paid: the operator reviews it while the shop sells.
func (s Submission) WaitsPaidFor() bool {
	return s.state == Waiting && s.paid
}

// OpenedFlag is the flag opened from the submission, once one was.
func (s Submission) OpenedFlag() (catalog.FlagID, bool) {
	return s.flag, s.state == Opened
}

// Facts is what the other modules hear of a verdict: an approval opens a paid flag, a rejection refunds it.
func (s Submission) Facts() []proto.Message {
	switch s.state {
	case Approved:
		return []proto.Message{&flagsv1.SubmissionApproved{
			SubmissionId: s.id.String(), ApprovedAt: timestamppb.New(s.decidedAt),
		}}
	case Rejected:
		return []proto.Message{&flagsv1.SubmissionRejected{
			SubmissionId: s.id.String(), RejectedAt: timestamppb.New(s.decidedAt),
		}}
	case Waiting, Opened:
	}
	return nil
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
		ID: s.id, Account: s.account, Flag: s.flag, Name: s.name, Pin: s.pin, Anthem: s.anthem.Library(),
		Picture: s.picture.TakeSnapshot(), OtherColors: s.others, Held: s.held, Paid: s.paid, Appealed: s.appealed,
		State:     s.state,
		Rejection: s.rejection, SubmittedAt: s.submittedAt, DecidedAt: s.decidedAt,
	}
}

type Snapshot struct {
	ID      SubmissionID
	Account catalog.AccountID
	// The flag opened from it; empty until then.
	Flag catalog.FlagID
	Name catalog.Name
	Pin  catalog.Pin
	// The library code of the anthem its flag takes, empty for none.
	Anthem      string
	Picture     pictures.Snapshot
	OtherColors pictures.OtherColors
	// The bucket holds the picture in private: until its approval, or 30 days after its rejection.
	Held bool
	// Time is paid and held for it until its flag opens: read from the payments' grants, never saved with it.
	Paid bool
	// Its player asked a person to look again: read from the appeals, never saved with it.
	Appealed bool
	State    State
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
	anthem, err := catalog.ParseAnthem(s.Anthem)
	if err != nil {
		return Submission{}, fmt.Errorf("the submission %s: %w", s.ID, err)
	}
	if err := s.checkState(); err != nil {
		return Submission{}, fmt.Errorf("the submission %s: %w", s.ID, err)
	}
	return Submission{
		id: s.ID, account: s.Account, flag: s.Flag, name: s.Name, pin: s.Pin, anthem: anthem, picture: picture,
		others: s.OtherColors, held: s.Held, paid: s.Paid, appealed: s.Appealed, state: s.State, rejection: s.Rejection,
		submittedAt: s.SubmittedAt, decidedAt: s.DecidedAt,
	}, nil
}

func (s Snapshot) checkState() error {
	if (s.State == Opened) != (s.Flag != "") {
		return fmt.Errorf("%w: %s, with the flag %q", ErrInvalidState, s.State, s.Flag)
	}
	switch s.State {
	case Waiting, Approved, Opened:
	case Rejected:
		if err := s.Rejection.check(); err != nil {
			return err
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
