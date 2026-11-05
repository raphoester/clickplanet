package create_upload_usecase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds/usecases/create_upload_usecase"
)

var rules = backgrounds.Rules{MaxBytes: 2 << 20}

func TestAnUploadDrawsAnIDAndSignsItsLink(t *testing.T) {
	out, err := create_upload_usecase.New(&backgrounds.SequentialIDs{}, rules, backgrounds.NewMemoryBucket()).Execute(
		create_upload_usecase.In{MediaType: "image/webp", Size: 300_000})

	require.NoError(t, err)
	assert.Equal(t, backgrounds.BackgroundID{15: 1}, out.ID)
	assert.Equal(t, "https://bucket.example/promo/backgrounds/"+out.ID.String()+"?size=300000", out.URL)
}

func TestAFileTheRulesRefuseGetsNoLink(t *testing.T) {
	_, err := create_upload_usecase.New(&backgrounds.SequentialIDs{}, rules, backgrounds.NewMemoryBucket()).Execute(
		create_upload_usecase.In{MediaType: "image/gif", Size: 300_000})

	require.ErrorIs(t, err, backgrounds.ErrInvalidBackground)
}

func TestABucketThatCannotSignIsTold(t *testing.T) {
	bucket := backgrounds.NewMemoryBucket().Failing(backgrounds.ErrBucketDown)

	_, err := create_upload_usecase.New(&backgrounds.SequentialIDs{}, rules, bucket).Execute(
		create_upload_usecase.In{MediaType: "image/png", Size: 300_000})

	require.ErrorIs(t, err, backgrounds.ErrBucketDown)
}
