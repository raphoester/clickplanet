package admission_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission"
)

func TestOnlyTheAdminPagesOriginMayCall(t *testing.T) {
	page := admission.NewPage("https://backoffice.clickplanet.lol")

	require.NoError(t, page.OriginError("https://backoffice.clickplanet.lol"))

	for _, origin := range []string{"", "https://clickplanet.lol", "http://backoffice.clickplanet.lol", "https://backoffice.clickplanet.lol.evil.example", "null"} {
		require.ErrorIs(t, page.OriginError(origin), admission.ErrForeignOrigin, origin)
	}
}
