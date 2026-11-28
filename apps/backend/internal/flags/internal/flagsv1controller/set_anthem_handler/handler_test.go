package set_anthem_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/set_anthem_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/set_anthem_handler"
)

type answering struct {
	err error
	in  set_anthem_usecase.In
}

func (a *answering) Execute(_ context.Context, in set_anthem_usecase.In) error {
	a.in = in
	return a.err
}

func TestTheAnthemChosenIsSetForTheFlag(t *testing.T) {
	useCase := &answering{}
	_, err := set_anthem_handler.New(useCase).SetAnthem(t.Context(), connect.NewRequest(&flagsv1.SetAnthemRequest{
		Flag: "r-jz9cwqzp0k", Anthem: &flagsv1.Anthem{Source: &flagsv1.Anthem_LibraryCode{LibraryCode: "de"}},
	}))
	require.NoError(t, err)
	assert.Equal(t, set_anthem_usecase.In{Flag: "r-jz9cwqzp0k", Anthem: "de"}, useCase.in)
}

func TestEachRefusalHasItsCode(t *testing.T) {
	for err, code := range map[error]connect.Code{
		catalog.ErrNotHolder:            connect.CodePermissionDenied,
		set_anthem_usecase.ErrNoAccount: connect.CodePermissionDenied,
		catalog.ErrRemoved:              connect.CodeFailedPrecondition,
		catalog.ErrInvalidAnthem:        connect.CodeInvalidArgument,
		catalog.ErrInvalidFlagID:        connect.CodeInvalidArgument,
		catalog.ErrUnknownFlag:          connect.CodeNotFound,
		errors.New("postgres is down"):  connect.CodeUnknown,
	} {
		_, got := set_anthem_handler.New(&answering{err: fmt.Errorf("wrapped: %w", err)}).
			SetAnthem(t.Context(), connect.NewRequest(&flagsv1.SetAnthemRequest{Flag: "r-jz9cwqzp0k"}))
		assert.Equal(t, code, connect.CodeOf(got), "%v", err)
	}
}
