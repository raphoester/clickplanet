package experiments_test

import (
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/experiments"
)

func TestEveryExperimentOnHasABriefAndTheLabHoldsThem(t *testing.T) {
	lab, err := experiments.Lab()
	require.NoError(t, err)

	require.Len(t, lab.Experiments, len(experiments.On))
	for i, experiment := range lab.Experiments {
		assert.Equal(t, experiments.On[i], string(experiment.Name))
		assert.NotEmpty(t, strings.TrimSpace(experiment.Brief))
	}
}

func TestEveryBriefIsOfAnExperimentOnOrOffAndNoneIsBoth(t *testing.T) {
	files, err := fs.Glob(os.DirFS("briefs"), "*.md")
	require.NoError(t, err)

	known := slices.Concat(experiments.On, experiments.Off)
	for _, file := range files {
		assert.Contains(t, known, strings.TrimSuffix(file, ".md"), "a brief no experiment names")
	}
	for _, name := range experiments.On {
		assert.NotContains(t, experiments.Off, name)
	}
}

func TestAnExperimentNoneNamesIsNoExperiment(t *testing.T) {
	_, err := experiments.Named("nobody")

	assert.Error(t, err)
}
