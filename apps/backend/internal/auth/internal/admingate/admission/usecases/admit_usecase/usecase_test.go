package admit_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission/usecases/admit_usecase"
)

const (
	adminOrigin = "https://backoffice.clickplanet.lol"
	approving   = "/auth.v1.AdminService/ApproveConnection"
)

type operators struct {
	asked []string
}

func (o *operators) Operator(_ context.Context, cookie string) (admission.Operator, error) {
	o.asked = append(o.asked, cookie)
	if cookie != "cp_sid=operator" {
		return "", admission.ErrNotAnOperator
	}
	return "the operator", nil
}

func (o *operators) Connection(_ context.Context, token string) (admission.Caller, error) {
	o.asked = append(o.asked, token)
	if token != "cpat_claude" {
		return admission.Caller{}, admission.ErrNoToken
	}
	return admission.ConnectionOf("the operator", "Claude"), nil
}

func page() admission.Page {
	return admission.NewPage(adminOrigin, "/auth.v1.AdminService/StartSignIn").WithSessionOnly(approving)
}

func TestTheOperatorOfTheAdminPageIsAdmitted(t *testing.T) {
	asked := &operators{}

	caller, err := admit_usecase.New(page(), asked).
		Execute(t.Context(), admission.Visit{Origin: adminOrigin, Cookie: "cp_sid=operator"})

	require.NoError(t, err)
	assert.Equal(t, admission.SessionOf("the operator"), caller)
}

func TestAnotherPageIsRefusedBeforeAuthIsAsked(t *testing.T) {
	asked := &operators{}

	_, err := admit_usecase.New(page(), asked).
		Execute(t.Context(), admission.Visit{Origin: "https://clickplanet.lol", Cookie: "cp_sid=operator"})

	require.ErrorIs(t, err, admission.ErrForeignOrigin)
	assert.Empty(t, asked.asked)
}

func TestSomebodyWhoDoesNotOperateIsRefused(t *testing.T) {
	_, err := admit_usecase.New(page(), &operators{}).
		Execute(t.Context(), admission.Visit{Origin: adminOrigin, Cookie: "cp_sid=player"})

	require.ErrorIs(t, err, admission.ErrNotAnOperator)
}

func TestTheSignInCallsNeedTheOriginAndNoSession(t *testing.T) {
	asked := &operators{}
	useCase := admit_usecase.New(page(), asked)

	caller, err := useCase.Execute(t.Context(), admission.Visit{Origin: adminOrigin, Procedure: "/auth.v1.AdminService/StartSignIn"})
	require.NoError(t, err)
	assert.Empty(t, caller.Operator(), "nobody is signed in yet")
	assert.Empty(t, asked.asked, "auth is not asked who calls")

	_, err = useCase.Execute(t.Context(), admission.Visit{Origin: "https://clickplanet.lol", Procedure: "/auth.v1.AdminService/StartSignIn"})
	require.ErrorIs(t, err, admission.ErrForeignOrigin, "the origin still holds")

	_, err = useCase.Execute(t.Context(), admission.Visit{Origin: adminOrigin, Procedure: "/planet.v1.AdminService/BanPlayer"})
	require.ErrorIs(t, err, admission.ErrNotAnOperator, "every other call needs a session")
}

func TestATokenIsAdmittedFromAnyOriginAsItsOperatorAndItsApp(t *testing.T) {
	for _, origin := range []string{"", "https://clickplanet.lol", adminOrigin} {
		for _, procedure := range []string{"/planet.v1.AdminService/TopPlayers", "/planet.v1.AdminService/BanPlayer"} {
			caller, err := admit_usecase.New(page(), &operators{}).Execute(t.Context(), admission.Visit{
				Origin: origin, Token: "cpat_claude", Procedure: procedure,
			})

			require.NoError(t, err, origin, procedure)
			assert.Equal(t, admission.ConnectionOf("the operator", "Claude"), caller, origin, procedure)
		}
	}
}

func TestATokenIsRefusedThePagesOwnCallsBeforeAuthIsAsked(t *testing.T) {
	for _, procedure := range []string{approving, "/auth.v1.AdminService/StartSignIn"} {
		asked := &operators{}

		_, err := admit_usecase.New(page(), asked).Execute(t.Context(), admission.Visit{
			Origin: adminOrigin, Cookie: "cp_sid=operator", Token: "cpat_claude", Procedure: procedure,
		})

		require.ErrorIs(t, err, admission.ErrSessionOnly, procedure)
		assert.Empty(t, asked.asked, procedure)
	}
}

func TestATokenNobodyHoldsIsRefusedAndTheCookieBesideItIsNotRead(t *testing.T) {
	asked := &operators{}

	_, err := admit_usecase.New(page(), asked).Execute(t.Context(), admission.Visit{
		Origin: adminOrigin, Cookie: "cp_sid=operator", Token: "cpat_made-up", Procedure: "/planet.v1.AdminService/TopPlayers",
	})

	require.ErrorIs(t, err, admission.ErrNoToken)
	assert.Equal(t, []string{"cpat_made-up"}, asked.asked, "a call that sends a token is judged by it alone")
}
