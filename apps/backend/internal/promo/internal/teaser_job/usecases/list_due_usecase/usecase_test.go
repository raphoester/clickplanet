package list_due_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job/usecases/list_due_usecase"
)

type promo struct {
	due []teaser_job.Due
}

func (p promo) Due(context.Context) ([]teaser_job.Due, error) {
	return p.due, nil
}

type store struct{ kept []teaser_job.Due }

func (s *store) Keep(_ context.Context, due teaser_job.Due) (string, error) {
	s.kept = append(s.kept, due)
	return due.FolderName(), nil
}

func TestEachBattlePromoSaysIsDueIsKeptForTheRenderer(t *testing.T) {
	ring, finale := teaser_job.Due{Battle: "entry-7"}, teaser_job.Due{Battle: "season-1/finale"}
	kept := &store{}

	due, err := list_due_usecase.New(promo{due: []teaser_job.Due{ring, finale}}, kept).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []list_due_usecase.Kept{{Folder: "entry-7", Due: ring}, {Folder: "season-1_finale", Due: finale}}, due)
	assert.Equal(t, []teaser_job.Due{ring, finale}, kept.kept)
}
