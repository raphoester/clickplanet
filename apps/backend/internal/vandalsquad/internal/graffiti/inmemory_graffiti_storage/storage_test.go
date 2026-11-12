package inmemory_graffiti_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/inmemory_graffiti_storage"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, &graffiti.StorageContractSuite{
		NewStorage: func() graffiti.Storage { return inmemory_graffiti_storage.New() },
		Kept: func(storage graffiti.Storage) []graffiti.Snapshot {
			return storage.(*inmemory_graffiti_storage.Storage).Kept() //nolint:forcetypeassert // NewStorage made it.
		},
	})
}
