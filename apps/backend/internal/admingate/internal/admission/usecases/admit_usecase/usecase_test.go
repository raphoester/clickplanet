package admit_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/admingate/internal/admission"
	"github.com/raphoester/clickplanet.lol-backend/internal/admingate/internal/admission/usecases/admit_usecase"
)

const adminOrigin = "https://admin.clickplanet.lol"

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

func TestTheOperatorOfTheAdminPageIsAdmitted(t *testing.T) {
	asked := &operators{}

	operator, err := admit_usecase.New(admission.NewPage(adminOrigin), asked).
		Execute(t.Context(), admission.Visit{Origin: adminOrigin, Cookie: "cp_sid=operator"})

	require.NoError(t, err)
	assert.Equal(t, admission.Operator("the operator"), operator)
}

func TestAnotherPageIsRefusedBeforeAuthIsAsked(t *testing.T) {
	asked := &operators{}

	_, err := admit_usecase.New(admission.NewPage(adminOrigin), asked).
		Execute(t.Context(), admission.Visit{Origin: "https://clickplanet.lol", Cookie: "cp_sid=operator"})

	require.ErrorIs(t, err, admission.ErrForeignOrigin)
	assert.Empty(t, asked.asked)
}

func TestSomebodyWhoDoesNotOperateIsRefused(t *testing.T) {
	_, err := admit_usecase.New(admission.NewPage(adminOrigin), &operators{}).
		Execute(t.Context(), admission.Visit{Origin: adminOrigin, Cookie: "cp_sid=player"})

	require.ErrorIs(t, err, admission.ErrNotAnOperator)
}
