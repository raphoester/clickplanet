package uuid_id_provider

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

type Provider struct{}

var _ reports.IDProvider = Provider{}

func (Provider) NewID() (reports.ReportID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return reports.ReportID{}, fmt.Errorf("failed to generate a uuidv7: %w", err)
	}
	return reports.ReportID(id), nil
}
