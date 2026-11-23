package flags_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/postgres_flag_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/list_flags_handler/flags_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestTheListIsTheFlagsTheGameShipsWithThenTheOnesOpenedByName(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	openedAt := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	closed, err := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Closed(openedAt.Add(time.Hour))
	require.NoError(t, err)
	require.NoError(t, postgres_flag_store.New(db).Save(t.Context(),
		closed, catalog.OpenFlag("r-0000000002", "Bavaria", openedAt)))

	list, err := flags_query.NewPostgresQuery(db).Flags(t.Context())
	require.NoError(t, err)

	builtIn := cpflags.BuiltIn()
	require.Len(t, list.GetFlags(), len(builtIn)+2)
	for i, flag := range builtIn {
		assert.True(t, proto.Equal(&flagsv1.Flag{Code: flag.Code, Name: flag.Name}, list.GetFlags()[i]), "%v", list.GetFlags()[i])
	}
	opened := list.GetFlags()[len(builtIn):]
	assert.True(t, proto.Equal(&flagsv1.Flag{Code: "r-0000000002", Name: "Bavaria"}, opened[0]), "%v", opened[0])
	assert.True(t, proto.Equal(&flagsv1.Flag{Code: "r-0000000001", Name: "Rennes", Closed: true}, opened[1]),
		"a closed flag stays in the list: %v", opened[1])
}
