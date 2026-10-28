package inmemory_flag_day_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/inmemory_flag_day_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, &flagmissions.StoreContractSuite{NewStore: func() flagmissions.Store { return inmemory_flag_day_store.New() }})
}
