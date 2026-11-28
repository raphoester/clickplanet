package flag_named_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations/usecases/name_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers/flag_named_subscriber"
)

type named struct {
	calls []name_flag_usecase.In
}

func (n *named) Execute(_ context.Context, in name_flag_usecase.In) error {
	n.calls = append(n.calls, in)
	return nil
}

var at = time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)

func TestAnOpeningAndARenameNameASoldFlag(t *testing.T) {
	useCase := &named{}

	require.NoError(t, flag_named_subscriber.NewOpened(useCase).Handle(t.Context(), &flagsv1.FlagOpened{
		Flag: "r-jz9cwqzp0k", Name: "Holy Roman Empire", OpenedAt: timestamppb.New(at),
	}))
	require.NoError(t, flag_named_subscriber.NewRenamed(useCase).Handle(t.Context(), &flagsv1.FlagRenamed{
		Flag: "r-jz9cwqzp0k", Name: "Heiliges Römisches Reich", RenamedAt: timestamppb.New(at.Add(time.Hour)),
	}))

	assert.Equal(t, []name_flag_usecase.In{
		{Code: "r-jz9cwqzp0k", Name: "Holy Roman Empire", At: at},
		{Code: "r-jz9cwqzp0k", Name: "Heiliges Römisches Reich", At: at.Add(time.Hour)},
	}, useCase.calls)
}

func TestAFactOfNoSoldFlagOrOfNoNameNamesNothing(t *testing.T) {
	useCase := &named{}

	require.NoError(t, flag_named_subscriber.NewOpened(useCase).Handle(t.Context(), &flagsv1.FlagOpened{
		Flag: "fr", Name: "France", OpenedAt: timestamppb.New(at),
	}))
	require.NoError(t, flag_named_subscriber.NewOpened(useCase).Handle(t.Context(), &flagsv1.FlagOpened{
		Flag: "r-jz9cwqzp0k", OpenedAt: timestamppb.New(at),
	}))

	assert.Empty(t, useCase.calls)
}
