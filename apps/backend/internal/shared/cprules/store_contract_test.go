package cprules_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &cprules.StoreContractSuite[*planetv1.Rules]{
		NewStore: func() cprules.Store[*planetv1.Rules] { return cprules.NewMemoryStore(parts) },
		Parts:    parts,
		Default:  seeded(),
		Rush:     func() *planetv1.Rules { return rush().Rules },
		Other: func() *planetv1.Rules {
			return &planetv1.Rules{Charges: &planetv1.ChargeRules{Shield: &planetv1.ShieldRules{PerTile: proto.Uint32(20)}}}
		},
	})
}

func TestTheMemoryChoicesKeepTheContract(t *testing.T) {
	suite.Run(t, &cprules.ChoiceStoreContractSuite{
		NewStore: func() cprules.Choices { return cprules.NewMemoryChoices() },
	})
}
