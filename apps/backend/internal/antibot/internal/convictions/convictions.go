package convictions

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
)

type Conviction struct {
	At        time.Time
	Scope     string
	Account   string
	Alongside string

	Flags       int
	Offence     int
	BannedUntil time.Time

	Readings []detect.Reading

	Clicks           int
	ActiveFor        time.Duration
	LongestGap       time.Duration
	TopCountry       string
	TopCountryClicks int
}

func Of(report detect.Report, at time.Time) Conviction {
	readings := make([]detect.Reading, 0, len(report.Opinions))
	for _, opinion := range report.Opinions {
		readings = append(readings, opinion.Reading())
	}

	return Conviction{
		At:               at,
		Scope:            report.Scope,
		Account:          report.Account,
		Alongside:        report.Alongside,
		Flags:            report.Flags,
		Offence:          report.Offence,
		BannedUntil:      report.BannedUntil,
		Readings:         readings,
		Clicks:           report.Clicks,
		ActiveFor:        report.ActiveFor,
		LongestGap:       report.LongestGap,
		TopCountry:       report.TopCountry,
		TopCountryClicks: report.TopCountryClicks,
	}
}

type Store interface {
	Keep(ctx context.Context, conviction Conviction) error

	// From since, until before until, oldest first.
	Between(ctx context.Context, since, until time.Time) ([]Conviction, error)
}
