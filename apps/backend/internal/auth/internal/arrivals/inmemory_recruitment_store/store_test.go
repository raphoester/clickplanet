package inmemory_recruitment_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/inmemory_recruitment_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, &arrivals.RecruitmentStoreContractSuite{
		NewStore: func() arrivals.Recruitments { return inmemory_recruitment_store.New() },
		Kept: func(store arrivals.Recruitments) []arrivals.RecruitmentSnapshot {
			return store.(*inmemory_recruitment_store.Store).Kept()
		},
	})
}
