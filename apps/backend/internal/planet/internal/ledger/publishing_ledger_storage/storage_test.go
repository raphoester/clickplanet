package publishing_ledger_storage_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/publishing_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var start = time.Date(2026, 9, 17, 23, 59, 0, 0, time.UTC)

func setUp() (publishing_ledger_storage.Storage, *inmemory_ledger_storage.Storage, *cpbootstrap.RecordedEvents) {
	inner := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(),
		slog.New(slog.DiscardHandler))
	events := cpbootstrap.NewRecordedEvents()
	return publishing_ledger_storage.New(inner, events), inner, events
}

func TestATakeWithAnAccountIsRecordedThenPublished(t *testing.T) {
	storage, inner, events := setUp()

	storage.Append(ledger.Taking{Tile: 42, Scope: "203.0.113.7", Account: account, Country: "fr", Previous: "de", At: start})

	var recorded []ledger.Taking
	inner.Replay(func(taking ledger.Taking) { recorded = append(recorded, taking) })
	require.Len(t, recorded, 1)

	published := events.Published()
	require.Len(t, published, 1)
	assert.True(t, proto.Equal(&planetv1.TileTaken{
		AccountId: account, TileId: 42, Country: "fr", TakenAt: timestamppb.New(start),
	}, published[0]))
}

func TestATakeWithNoAccountIsRecordedAndNotPublished(t *testing.T) {
	storage, inner, events := setUp()

	storage.Append(ledger.Taking{Tile: 42, Scope: "203.0.113.7", Country: "fr", At: start})

	takes := 0
	inner.Replay(func(ledger.Taking) { takes++ })
	assert.Equal(t, 1, takes)
	assert.Empty(t, events.Published())
}
