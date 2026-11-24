package submissions_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &submissions.StoreContractSuite{
		NewStore: func() submissions.Store { return submissions.NewMemoryStore() },
	})
}
