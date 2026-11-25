package experiments_test

import (
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
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

func TestTheCarouselIsDrawnAsPhotosAndTheOthersAsClips(t *testing.T) {
	carousel, err := experiments.Named("carousel")
	require.NoError(t, err)
	assert.Equal(t, posts.Carousel, carousel.Format)

	for _, name := range []string{"far-invasion", "call-to-arms", "war-report"} {
		experiment, err := experiments.Named(name)
		require.NoError(t, err)
		assert.Equal(t, posts.GlobeReplay, experiment.Format, name)
	}
}

func TestTheWarReportAloneIsToldByAVoice(t *testing.T) {
	for _, name := range slices.Concat(experiments.On, experiments.Off) {
		experiment, err := experiments.Named(name)
		require.NoError(t, err)
		assert.Equal(t, name == "war-report", experiment.Narrated, name)
	}
}
