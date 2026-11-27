package appeals_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &appeals.StoreContractSuite{
		NewStore: func() appeals.Store { return appeals.NewMemoryStore() },
		Seed:     func(context.Context, submissions.Snapshot) error { return nil },
	})
}
