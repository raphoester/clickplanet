package keep_played_flag_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog/usecases/keep_played_flag_usecase"
)

const rennes = "r-k3p9x2ma0b"

var (
	opened = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	played = opened.Add(time.Minute)
)

type commands struct {
	sent    []proto.Message
	failing error
}

func (c *commands) Send(_ context.Context, commands ...proto.Message) error {
	if c.failing != nil {
		return c.failing
	}
	c.sent = append(c.sent, commands...)
	return nil
}

func newCatalog(t *testing.T) catalog.Catalog {
	t.Helper()
	flags := catalog.NewCatalog(catalog.NewMemoryStore(), catalog.Unlocked)
	require.NoError(t, flags.Open(t.Context(), rennes, opened))
	return flags
}

func TestTheFlagsModuleIsAskedOnceToKeepAFlagPlayed(t *testing.T) {
	sent := &commands{}
	useCase := keep_played_flag_usecase.New(newCatalog(t), sent)

	require.NoError(t, useCase.Execute(t.Context(), keep_played_flag_usecase.In{Flag: rennes, PlayedAt: played}))
	require.NoError(t, useCase.Execute(t.Context(), keep_played_flag_usecase.In{Flag: rennes, PlayedAt: played.Add(time.Hour)}))

	require.Len(t, sent.sent, 1)
	assert.True(t, proto.Equal(&flagsv1.KeepFlag{Flag: rennes, PlayedAt: timestamppb.New(played)}, sent.sent[0]), "%v", sent.sent[0])
}

func TestAFlagTheGameShipsWithOrNeverHeardIsNeverAsked(t *testing.T) {
	sent := &commands{}
	useCase := keep_played_flag_usecase.New(newCatalog(t), sent)

	require.NoError(t, useCase.Execute(t.Context(), keep_played_flag_usecase.In{Flag: "fr", PlayedAt: played}))
	require.NoError(t, useCase.Execute(t.Context(), keep_played_flag_usecase.In{Flag: "r-0000000001", PlayedAt: played}))

	assert.Empty(t, sent.sent)
}

func TestAFlagNotAskedForIsAskedForAgain(t *testing.T) {
	sent := &commands{failing: errors.New("redis is down")}
	flags := newCatalog(t)
	useCase := keep_played_flag_usecase.New(flags, sent)

	require.Error(t, useCase.Execute(t.Context(), keep_played_flag_usecase.In{Flag: rennes, PlayedAt: played}))
	unkept, err := flags.Unkept(t.Context(), rennes)
	require.NoError(t, err)
	assert.True(t, unkept, "noted only once sent")

	sent.failing = nil
	require.NoError(t, useCase.Execute(t.Context(), keep_played_flag_usecase.In{Flag: rennes, PlayedAt: played}))
	assert.Len(t, sent.sent, 1)
}
