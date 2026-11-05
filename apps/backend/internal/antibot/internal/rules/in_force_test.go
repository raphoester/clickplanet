package rules_test

import (
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/cohort"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/jury"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

func TestAReaderReadsItsPartOfTheSetEnforcedLast(t *testing.T) {
	inForce := rules.NewInForce()
	crowd := rules.Reader(inForce, cohort.Part)
	suspects := rules.Reader(inForce, jury.Part)

	for _, kept := range []rules.Snapshot{
		rules.Seeded(),
		rules.Named("rush", func(r *antibotv1.Rules) { r.Cohort.CertainMembers = proto.Uint32(50) }),
	} {
		set, err := kept.Restore(guard.Parts(), rules.Seeded())
		require.NoError(t, err)
		inForce.Enforce(set)
	}

	assert.Equal(t, rules.SetName("rush"), inForce.Name())
	assert.Equal(t, 50, crowd().CertainMembers)
	assert.Equal(t, 2, suspects().MinSuspects)
}
