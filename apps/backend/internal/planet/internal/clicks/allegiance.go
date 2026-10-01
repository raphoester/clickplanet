package clicks

import (
	"math"
	"time"
)

const (
	// A take counts half as much every flagHalfLife, so the main flag is about the last day's.
	flagHalfLife = 12 * time.Hour

	// Past it every take counts 1/64 or less, and the allegiance is forgotten.
	flagMemory = 6 * flagHalfLife

	// A country whose takes have faded below one take in 64 is dropped from the tally.
	fadedWeight = 1.0 / 64
)

// AllegianceKey names one tally. The store keeps it opaque; only the names below say whose it is.
type AllegianceKey string

func AccountAllegianceKey(account string) AllegianceKey { return AllegianceKey("account:" + account) }

func ScopeAllegianceKey(scope string) AllegianceKey { return AllegianceKey("scope:" + scope) }

// Allegiance is the flag a player takes tiles for most, so one click for another flag does not change it.
type Allegiance struct {
	weights map[string]float64
	at      time.Time
}

// With is a copy that counts one more take for country, at now.
func (a Allegiance) With(country string, now time.Time) Allegiance {
	fade := 1.0
	if elapsed := now.Sub(a.at); elapsed > 0 {
		fade = math.Exp2(-elapsed.Seconds() / flagHalfLife.Seconds())
	}

	weights := make(map[string]float64, len(a.weights)+1)
	for flag, weight := range a.weights {
		if faded := weight * fade; faded >= fadedWeight {
			weights[flag] = faded
		}
	}
	weights[country]++

	return Allegiance{weights: weights, at: now}
}

// Flag is the country with the most weight, "" for none; a tie goes to the code that sorts first.
func (a Allegiance) Flag() string {
	main, most := "", 0.0
	for flag, weight := range a.weights {
		if weight > most || (weight == most && flag < main) {
			main, most = flag, weight
		}
	}

	return main
}

// Faded says whether the last take is older than flagMemory.
func (a Allegiance) Faded(now time.Time) bool {
	return now.Sub(a.at) > flagMemory
}
