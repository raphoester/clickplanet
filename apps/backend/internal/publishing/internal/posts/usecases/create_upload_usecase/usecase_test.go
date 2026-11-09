package create_upload_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/usecases/create_upload_usecase"
)

type uploads struct {
	asked []posts.Media
}

func (u *uploads) MediaUpload(_ context.Context, media posts.Media) (posts.MediaUpload, error) {
	u.asked = append(u.asked, media)
	return posts.MediaUpload{UploadURL: "https://storage.example/clip-1.mp4?signature=s", Media: "https://media.example/clip-1.mp4"}, nil
}

func TestAnMP4IsGivenALinkByTheService(t *testing.T) {
	service := &uploads{}
	media := posts.Media{FileName: "clip-1.mp4", ContentType: posts.VideoMP4, Size: 80 << 20}

	upload, err := create_upload_usecase.New(service).Execute(t.Context(), media)

	require.NoError(t, err)
	assert.Equal(t, posts.MediaURL("https://media.example/clip-1.mp4"), upload.Media)
	assert.Equal(t, []posts.Media{media}, service.asked)
}

func TestAFileThatCannotBePostedAsksTheServiceNothing(t *testing.T) {
	service := &uploads{}

	_, err := create_upload_usecase.New(service).Execute(t.Context(), posts.Media{FileName: "clip.png", ContentType: "image/png", Size: 1})

	require.ErrorIs(t, err, posts.ErrInvalidMedia)
	assert.Empty(t, service.asked)
}
