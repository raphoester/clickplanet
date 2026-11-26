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
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/list_flags_handler/flags_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestTheListIsTheFlagsTheGameShipsWithThenTheOnesOpenedByName(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	openedAt := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	removed, err := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Removed("Broke the rules.", openedAt.Add(time.Hour))
	require.NoError(t, err)
	ended, ok := catalog.OpenFlag("r-0000000003", "Savoy", openedAt).Expired(openedAt.AddDate(0, 2, 0))
	require.True(t, ok)
	require.NoError(t, postgres_flag_store.New(db).Save(t.Context(),
		removed, catalog.OpenFlag("r-0000000002", "Bavaria", openedAt), ended))

	list, err := flags_query.NewPostgresQuery(db, flagmessage.New("https://media.test")).Flags(t.Context())
	require.NoError(t, err)

	builtIn := cpflags.BuiltIn()
	require.Len(t, list.GetFlags(), len(builtIn)+7)
	for i, flag := range builtIn {
		assert.True(t, proto.Equal(&flagsv1.Flag{Code: flag.Code, Name: flag.Name}, list.GetFlags()[i]), "%v", list.GetFlags()[i])
	}
	opened := list.GetFlags()[len(builtIn):]
	assert.Equal(t, [2]string{"r-0000000002", "Bavaria"}, [2]string{opened[0].GetCode(), opened[0].GetName()})
	assert.False(t, opened[0].GetClosed())
	assert.Nil(t, opened[0].GetSold().GetClosedAt())
	assert.Equal(t, "https://media.test/"+catalog.TestPicture.Key(), opened[0].GetSold().GetPicture().GetUrl())
	assert.Equal(t, uint32(300), opened[0].GetSold().GetPicture().GetWidth())
	assert.Equal(t, openedAt.Add(30*24*time.Hour), opened[0].GetSold().GetEndsAt().AsTime())
	for i, left := range map[int][2]string{1: {"xb", "Brittany"}, 2: {"gb-eng", "England"}, 5: {"gb-sct", "Scotland"}, 6: {"gb-wls", "Wales"}} {
		listed := [2]string{opened[i].GetCode(), opened[i].GetName()}
		assert.Equal(t, left, listed, "sold since it left the build")
		assert.NotNil(t, opened[i].GetSold().GetPicture(), left)
	}
	assert.Equal(t, [2]string{"r-0000000001", "Rennes"}, [2]string{opened[3].GetCode(), opened[3].GetName()})
	assert.True(t, opened[3].GetClosed(), "a removed flag stays in the list")
	assert.True(t, opened[3].GetSold().GetRemoved())
	assert.Equal(t, openedAt.Add(time.Hour), opened[3].GetSold().GetClosedAt().AsTime())
	assert.Equal(t, [2]string{"r-0000000003", "Savoy"}, [2]string{opened[4].GetCode(), opened[4].GetName()})
	assert.True(t, opened[4].GetClosed())
	assert.False(t, opened[4].GetSold().GetRemoved(), "a flag whose time ended is free to claim")
	assert.Equal(t, openedAt.Add(30*24*time.Hour), opened[4].GetSold().GetClosedAt().AsTime())
}
