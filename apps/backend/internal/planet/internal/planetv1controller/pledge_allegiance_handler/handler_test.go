package pledge_allegiance_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/pledge_allegiance_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const ada = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

type stubBook struct {
	err   error
	flags []string
	payer clicks.Payer
}

func (s *stubBook) Pledge(_ context.Context, payer clicks.Payer, flag string) error {
	s.payer = payer
	s.flags = append(s.flags, flag)
	return s.err
}

func pledge(t *testing.T, book *stubBook, flag string) error {
	t.Helper()
	ctx := cpctx.AddAccountToContext(t.Context(), ada)
	req := connect.NewRequest(&planetv1.PledgeAllegianceRequest{CountryId: flag})
	_, err := pledge_allegiance_handler.New(book).PledgeAllegiance(ctx, req)
	return err //nolint:wrapcheck // the test reads the handler's own error.
}

func TestAPledgeIsForTheCallersAccount(t *testing.T) {
	book := &stubBook{}

	require.NoError(t, pledge(t, book, "de"))

	assert.Equal(t, []string{"de"}, book.flags)
	assert.Equal(t, ada, book.payer.Account)
}

func TestAPledgeRefusedSaysWhichFlagAndUntilWhen(t *testing.T) {
	until := time.Date(2026, 10, 23, 17, 0, 0, 0, time.UTC)
	book := &stubBook{err: fmt.Errorf("wrapped: %w", allegiances.Refusal{Flag: "fr", Until: until, Reason: allegiances.ErrChangedTooSoon})}

	err := pledge(t, book, "de")

	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	var refused *connect.Error
	require.ErrorAs(t, err, &refused)
	require.Len(t, refused.Details(), 1)
	value, err := refused.Details()[0].Value()
	require.NoError(t, err)
	assert.True(t, proto.Equal(&planetv1.Allegiance{CountryId: "fr", ChangeableAtUnixMs: until.UnixMilli()}, value))
}

func TestAPledgeWithNoAccountOrNoCountryIsTheCallersFault(t *testing.T) {
	err := pledge(t, &stubBook{err: allegiances.ErrNoAccount}, "de")
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	err = pledge(t, &stubBook{err: fmt.Errorf("%q: %w", "zz", clicks.ErrUnknownCountry)}, "zz")
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAPledgeThatFailsIsLeftToTheErrorNet(t *testing.T) {
	down := errors.New("postgres is down")

	require.ErrorIs(t, pledge(t, &stubBook{err: down}, "de"), down)
}
