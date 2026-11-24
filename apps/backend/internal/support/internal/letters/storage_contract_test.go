package letters_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
)

func TestTheMemoryStorageKeepsTheContract(t *testing.T) {
	suite.Run(t, &letters.StorageContractSuite{
		NewStorage: func() letters.Storage { return letters.NewMemoryStorage() },
	})
}
