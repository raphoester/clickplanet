package inmemory_allegiance_store_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_allegiance_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	clicks.AllegianceStorageContractSuite
}

func (s *testSuite) SetupSuite() {
	s.NewStorage = func() clicks.AllegianceStorage { return inmemory_allegiance_store.New() }
}

func (s *testSuite) TestAFailingStoreFailsEveryCall() {
	store := inmemory_allegiance_store.New()
	cause := errors.New("postgres is down")
	store.FailWith(cause)

	_, err := store.Allegiances(s.T().Context(), clicks.AccountAllegianceKey("ada"))
	s.Require().ErrorIs(err, cause)
	s.Require().ErrorIs(store.SaveAllegiances(s.T().Context(), map[clicks.AllegianceKey]clicks.Allegiance{}), cause)
	_, err = store.DeleteAllegiancesBefore(s.T().Context(), time.Now())
	s.Require().ErrorIs(err, cause)
}
