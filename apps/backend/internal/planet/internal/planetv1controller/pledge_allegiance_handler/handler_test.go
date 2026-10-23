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

type stubQuery struct {
	held    *planetv1.Allegiance
	err     error
	account allegiances.Account
}

func (s *stubQuery) Allegiance(_ context.Context, account allegiances.Account) (*planetv1.GetAllegianceResponse, error) {
	s.account = account
	return &planetv1.GetAllegianceResponse{Allegiance: s.held}, s.err
}

func pledge(t *testing.T, book *stubBook, query *stubQuery, flag string) (*connect.Response[planetv1.PledgeAllegianceResponse], error) {
	t.Helper()
	ctx := cpctx.AddAccountToContext(t.Context(), ada)
	req := connect.NewRequest(&planetv1.PledgeAllegianceRequest{CountryId: flag})
	return pledge_allegiance_handler.New(book, query).PledgeAllegiance(ctx, req) //nolint:wrapcheck // the test reads the handler's own error.
}

func TestAPledgeAnswersTheFlagNowHeld(t *testing.T) {
	book := &stubBook{}
	query := &stubQuery{held: &planetv1.Allegiance{CountryId: "de", ChangeableAtUnixMs: 1_700_000_000_000}}

	res, err := pledge(t, book, query, "de")

	require.NoError(t, err)
	assert.Equal(t, []string{"de"}, book.flags)
	assert.Equal(t, ada, book.payer.Account)
	assert.Equal(t, allegiances.Account(ada), query.account)
	assert.True(t, proto.Equal(query.held, res.Msg.GetAllegiance()))
}

func TestAPledgeRefusedSaysWhichFlagAndUntilWhen(t *testing.T) {
	until := time.Date(2026, 10, 23, 17, 0, 0, 0, time.UTC)
	book := &stubBook{err: fmt.Errorf("wrapped: %w", allegiances.Refusal{Flag: "fr", Until: until})}

	_, err := pledge(t, book, &stubQuery{}, "de")

	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	var refused *connect.Error
	require.ErrorAs(t, err, &refused)
	require.Len(t, refused.Details(), 1)
	value, err := refused.Details()[0].Value()
	require.NoError(t, err)
	assert.True(t, proto.Equal(&planetv1.Allegiance{CountryId: "fr", ChangeableAtUnixMs: until.UnixMilli()}, value))
}

func TestAPledgeThatFailsIsLeftToTheErrorNet(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := pledge(t, &stubBook{err: down}, &stubQuery{}, "de")
	require.ErrorIs(t, err, down)

	_, err = pledge(t, &stubBook{}, &stubQuery{err: down}, "de")
	require.ErrorIs(t, err, down)
}
