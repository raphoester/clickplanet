package arrivals

import (
	"context"
	"fmt"
)

// Who recruited a recruit's arrival: a table of its own, so no other arrival has a recruiter to leave empty.
type Recruitment struct {
	arrival   ArrivalID
	recruiter AccountID
}

// The recruitment with nothing that says who recruited: the arrival it is of stays a recruited one.
func (r Recruitment) Anonymized() Recruitment {
	r.recruiter = AccountID{}
	return r
}

func AnonymizedRecruitments(kept []Recruitment) []Recruitment {
	anonymized := make([]Recruitment, 0, len(kept))
	for _, recruitment := range kept {
		anonymized = append(anonymized, recruitment.Anonymized())
	}
	return anonymized
}

func (r Recruitment) TakeSnapshot() RecruitmentSnapshot {
	return RecruitmentSnapshot{Arrival: r.arrival, Recruiter: r.recruiter}
}

type RecruitmentSnapshot struct {
	Arrival ArrivalID
	// Zero once the recruiter's account is deleted.
	Recruiter AccountID
}

func (s RecruitmentSnapshot) Restore() (Recruitment, error) {
	if s.Arrival == (ArrivalID{}) {
		return Recruitment{}, fmt.Errorf("%w: a recruitment of no arrival", ErrInvalidArrival)
	}
	return Recruitment{arrival: s.Arrival, recruiter: s.Recruiter}, nil
}

// A store finds recruitments and saves them. It holds no rule: what changes a recruitment is a method of Recruitment.
type Recruitments interface {
	// A recruitment saved again replaces itself. All of them, or none.
	Save(ctx context.Context, recruitments ...Recruitment) error
	// The recruitments that still name the recruiter, by arrival.
	By(ctx context.Context, recruiter AccountID) ([]Recruitment, error)
}
