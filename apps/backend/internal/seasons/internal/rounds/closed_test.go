package rounds_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
)

func TestAClosedRoundCarriesItsNumberAndItsResults(t *testing.T) {
	held := map[rounds.Country]uint64{"de": 20, "fr": 30}

	assert.Equal(t, rounds.Closed{Round: day, Number: 5, Results: day.Results(held)}, day.Closed(5, held))
}
