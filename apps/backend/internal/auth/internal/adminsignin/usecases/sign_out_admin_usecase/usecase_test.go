package sign_out_admin_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin/usecases/sign_out_admin_usecase"
)

type deleter struct {
	deleted []accounts.TokenHash
}

func (d *deleter) DeleteSession(_ context.Context, tokenHash accounts.TokenHash) error {
	d.deleted = append(d.deleted, tokenHash)
	return nil
}

func TestSigningOutDeletesTheAdminSessionAndOnlyIt(t *testing.T) {
	kept := &deleter{}

	require.NoError(t, sign_out_admin_usecase.New(kept).Execute(t.Context(), "cp_sid=game; cp_admin=admin"))

	assert.Equal(t, []accounts.TokenHash{accounts.TokenOf("admin").Hash()}, kept.deleted, "the game session stays")
}

func TestABrowserWithNoAdminSessionSignsOutOfNothing(t *testing.T) {
	kept := &deleter{}

	require.NoError(t, sign_out_admin_usecase.New(kept).Execute(t.Context(), "cp_sid=game"))

	assert.Empty(t, kept.deleted)
}
