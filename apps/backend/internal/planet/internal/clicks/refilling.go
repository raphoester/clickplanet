package clicks

import (
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type Limiter interface {
	Peek(key cpratelimit.Key) cpratelimit.State
	Fill(key cpratelimit.Key) (bool, cpratelimit.State)
	Add(key cpratelimit.Key, n float64)
}

func NewRefilling(limiter Limiter, buckets Buckets) Refilling {
	return Refilling{limiter: limiter, buckets: buckets}
}

type Refilling struct {
	limiter Limiter
	buckets Buckets
}

func (r Refilling) Full(payer Payer) bool {
	return r.lack(payer) <= 0
}

// Refill fills the payer's bank, and gives the buckets it shares with its scope what that gave, never more.
func (r Refilling) Refill(payer Payer) {
	lack := r.lack(payer)
	for _, key := range r.buckets.Bank(payer) {
		_, _ = r.limiter.Fill(key)
	}
	for _, key := range r.buckets.Ceiling(payer) {
		r.limiter.Add(key, lack)
	}
}

func (r Refilling) Budget(payer Payer, price Price) Budget {
	keys := r.buckets.Keys(payer, price)
	states := make([]cpratelimit.State, len(keys))
	for i, key := range keys {
		states[i] = r.limiter.Peek(key)
	}

	return r.buckets.BudgetOf(payer, states, price)
}

// lack is how far what the payer can spend is from a full bank: the size of its own, less the emptiest bucket it spends from.
func (r Refilling) lack(payer Payer) float64 {
	size, spendable := math.Inf(1), math.Inf(1)
	for _, key := range r.buckets.Bank(payer) {
		state := r.limiter.Peek(key)
		size, spendable = min(size, float64(state.Capacity)), min(spendable, state.Tokens)
	}
	for _, key := range r.buckets.Ceiling(payer) {
		spendable = min(spendable, r.limiter.Peek(key).Tokens)
	}

	return size - spendable
}
