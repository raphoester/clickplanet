package eras_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &eras.StoreContractSuite{NewStore: func() eras.Store { return eras.NewMemoryStore() }})
}
