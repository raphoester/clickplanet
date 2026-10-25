package rules

import "context"

type LayerStore interface {
	// Baseline is the baseline kept for this request, or none.
	Baseline(ctx context.Context, request RequestID) (Baseline, error)

	// LatestBaseline is the baseline applied last, or none.
	LatestBaseline(ctx context.Context) (Baseline, error)

	SaveBaseline(ctx context.Context, baseline Baseline) error

	// Override is the override set last, running or ended, or none.
	Override(ctx context.Context) (Override, error)

	SaveOverride(ctx context.Context, override Override) error
}
