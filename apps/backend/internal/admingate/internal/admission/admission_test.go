package admission_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/admingate/internal/admission"
)

func TestOnlyTheAdminPagesOriginMayCall(t *testing.T) {
	page := admission.NewPage("https://admin.clickplanet.lol")

	require.NoError(t, page.OriginError("https://admin.clickplanet.lol"))

	for _, origin := range []string{"", "https://clickplanet.lol", "http://admin.clickplanet.lol", "https://admin.clickplanet.lol.evil.example", "null"} {
		require.ErrorIs(t, page.OriginError(origin), admission.ErrForeignOrigin, origin)
	}
}
