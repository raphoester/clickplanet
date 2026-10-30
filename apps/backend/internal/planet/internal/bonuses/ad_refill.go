package bonuses

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrAdRefillUsed = errors.New("the bank had its ad refill in this span")

var (
	adRefillAccounts = uuid.MustParse("13fae780-492f-40ca-97cc-d16d49f53781")
	adRefillNetworks = uuid.MustParse("034b0e8c-9c0c-41ec-b8e2-78fa09a61fdf")
)

type AdRefillRefusal struct {
	Until time.Time
}

func (r AdRefillRefusal) Error() string {
	return fmt.Sprintf("%s: the next from %s", ErrAdRefillUsed, r.Until.UTC().Format(time.RFC3339))
}

func (r AdRefillRefusal) Unwrap() error { return ErrAdRefillUsed }

type Grants interface {
	Record(ctx context.Context, grant GrantID, at time.Time) (bool, error)
	Recorded(ctx context.Context, grant GrantID) (bool, error)
}

func NewAdRefills(grants Grants, rules func() Rules, clock cptime.Clock) AdRefills {
	return AdRefills{grants: grants, rules: rules, clock: clock}
}

type AdRefills struct {
	grants Grants
	rules  func() Rules
	clock  cptime.Clock
}

func (a AdRefills) AvailableAt(ctx context.Context, payer clicks.Payer) (time.Time, error) {
	if HolderOf(payer) == NoHolder {
		return time.Time{}, nil
	}

	span := a.spanOf(a.clock.Now())
	for _, grant := range span.grants(payer) {
		recorded, err := a.grants.Recorded(ctx, grant)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to read the ad refill %s: %w", grant, err)
		}
		if recorded {
			return span.end, nil
		}
	}
	return span.start, nil
}

// Record keeps every grant of the payer's bank in the span, or none: run it in one transaction.
func (a AdRefills) Record(ctx context.Context, payer clicks.Payer) error {
	at := a.clock.Now()
	span := a.spanOf(at)
	for _, grant := range span.grants(payer) {
		first, err := a.grants.Record(ctx, grant, at)
		if err != nil {
			return fmt.Errorf("failed to record the ad refill %s: %w", grant, err)
		}
		if !first {
			return AdRefillRefusal{Until: span.end}
		}
	}
	return nil
}

type adSpan struct {
	start, end time.Time
}

func (a AdRefills) spanOf(at time.Time) adSpan {
	every := a.rules().AdRefill.Every
	start := time.Unix(0, at.UnixNano()/int64(every)*int64(every)).UTC()
	return adSpan{start: start, end: start.Add(every)}
}

// The guests behind one network spend from one bank, so a guest's network has its grant too.
func (s adSpan) grants(payer clicks.Payer) []GrantID {
	grants := []GrantID{s.grantOf(adRefillAccounts, string(HolderOf(payer)))}
	if !payer.Linked {
		grants = append(grants, s.grantOf(adRefillNetworks, payer.Scope))
	}
	return grants
}

// Named by its start, so a new `every` gives no second refill in a span that starts with the old one.
func (s adSpan) grantOf(space uuid.UUID, bank string) GrantID {
	return GrantID(uuid.NewSHA1(space, fmt.Appendf(nil, "%s %d", bank, s.start.UnixNano())))
}
