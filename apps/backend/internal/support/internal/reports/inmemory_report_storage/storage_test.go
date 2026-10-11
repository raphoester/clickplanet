package inmemory_report_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/inmemory_report_storage"
)

func TestTheContract(t *testing.T) {
	storage := inmemory_report_storage.New()

	suite.Run(t, &reports.StorageContractSuite{
		NewStorage: func() reports.Storage {
			storage = inmemory_report_storage.New()
			return storage
		},
		Kept: func(reports.Storage) []reports.Snapshot { return storage.Kept() },
	})
}
