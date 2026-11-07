package inmemory_judgement_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/inmemory_judgement_storage"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, &judgements.StorageContractSuite{
		NewStorage: func() judgements.Storage { return inmemory_judgement_storage.New() },
	})
}
