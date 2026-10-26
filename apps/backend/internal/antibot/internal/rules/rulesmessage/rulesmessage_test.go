package rulesmessage_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules/rulesmessage"
)

func TestASetReadsBackAsItWasWritten(t *testing.T) {
	kept := rules.Named("rush", func(s *rules.Snapshot) {
		skew, coherence := -0.1, 0.9
		s.Metronome.Shape.MaxSkew = &skew
		s.Metronome.Clock.CertainCoherence = &coherence
	})

	document, err := rulesmessage.Document.Marshal(rulesmessage.Of(kept))
	require.NoError(t, err)

	var read antibotv1.Rules
	require.NoError(t, protojson.Unmarshal(document, &read))

	assert.Equal(t, kept, rulesmessage.SnapshotOf("rush", &read))
}

func TestTheDocumentWritesEveryZero(t *testing.T) {
	document, err := rulesmessage.Document.Marshal(rulesmessage.Of(rules.Seeded()))
	require.NoError(t, err)

	var written struct {
		Retaker   map[string]any
		Metronome struct{ Shape map[string]any }
	}
	require.NoError(t, json.Unmarshal(document, &written))

	assert.Contains(t, written.Retaker, "certainTiles")
	assert.NotContains(t, written.Metronome.Shape, "maxSkew", "an unset bound never reads, and is no zero")
}

func TestAPartLeftOutReadsAsZerosTheSetRefuses(t *testing.T) {
	message := rulesmessage.Of(rules.Seeded())
	message.Rotator = nil

	_, err := rulesmessage.SnapshotOf(rules.Default, message).Restore()

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Contains(t, err.Error(), "rotator.window is 0s")
}
