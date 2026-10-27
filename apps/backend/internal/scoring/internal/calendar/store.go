package calendar

import "context"

// Store keeps the decision told last on each season.
type Store interface {
	// Decision is false when none is kept on the season.
	Decision(ctx context.Context, number Number) (Decision, bool, error)
	Decisions(ctx context.Context) ([]Decision, error)
	Save(ctx context.Context, decision Decision) error
}

// WindowsOf are the decisions' windows.
func WindowsOf(decisions []Decision) []Window {
	windows := make([]Window, 0, len(decisions))
	for _, decision := range decisions {
		windows = append(windows, decision.Window)
	}
	return windows
}
