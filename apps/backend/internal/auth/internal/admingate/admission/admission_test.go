package admission_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestATokenMakesNoneOfThePagesOwnCalls(t *testing.T) {
	page := admission.NewPage("https://backoffice.clickplanet.lol", "/auth.v1.AdminService/StartSignIn").
		WithSessionOnly("/auth.v1.AdminService/ApproveConnection")

	for _, procedure := range []string{"/auth.v1.AdminService/StartSignIn", "/auth.v1.AdminService/ApproveConnection"} {
		require.ErrorIs(t, page.TokenError(procedure), admission.ErrSessionOnly, procedure)
	}
	require.NoError(t, page.TokenError("/planet.v1.AdminService/BanPlayer"))
	assert.False(t, page.SignIn("/auth.v1.AdminService/ApproveConnection"), "letting an app in still needs the session")
}

func TestACallerIsTheOperatorAndTheAppItConnected(t *testing.T) {
	connected := admission.ConnectionOf("ada", "Claude")
	assert.Equal(t, admission.Operator("ada"), connected.Operator())
	assert.Equal(t, "Claude", connected.Client())

	session := admission.SessionOf("ada")
	assert.Equal(t, admission.Operator("ada"), session.Operator())
	assert.Empty(t, session.Client(), "the page's session is no app")
}
