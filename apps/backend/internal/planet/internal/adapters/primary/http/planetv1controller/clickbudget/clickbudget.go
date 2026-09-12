// Package clickbudget is one function: how a limiter reading becomes the
// ClickBudget message. Two procedures answer with one — the click that just
// spent a token, and the cold-start read — so the shape has to be agreed
// somewhere, and a package with a single job is a smaller thing to agree on
// than a shared bag of edge helpers.
package clickbudget

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

// Encode puts a limiter reading on the wire. Capacity and refill rate go with
// it: the client redraws the allowance many times a second, and it can only do
// that without asking again if it knows the policy it must replay. Changing
// rateLimiter.* therefore changes the display with no frontend release.
func Encode(state cpratelimit.State) *planetv1.ClickBudget {
	return &planetv1.ClickBudget{
		// A refused caller is at or below zero; the meter shows empty, not negative.
		Tokens:          max(state.Tokens, 0),
		Capacity:        uint32(state.Capacity),
		RefillPerSecond: state.PerSecond,
	}
}
