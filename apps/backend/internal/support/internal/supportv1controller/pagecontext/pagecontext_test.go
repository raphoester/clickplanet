package pagecontext_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/pagecontext"
)

func TestEverythingThePageSaidIsHandedOnWithTheBrowserTheServerRead(t *testing.T) {
	header := http.Header{}
	header.Set("User-Agent", "Mozilla/5.0 (iPhone)")

	said := pagecontext.Diagnostics(&supportv1.PageContext{
		Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, ConnectionDown: true, PixelRatio: 3,
		GpuVendor: "Apple Inc.", GpuRenderer: "Apple GPU", WebglVersion: "WebGL 2.0", MaxTextureSize: 4096,
		ContextLost: true, Cores: 6, MemoryGb: 4,
	}, header)

	assert.Equal(t, notes.Diagnostics{
		Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, PixelRatio: 3, ConnectionDown: true,
		UserAgent: "Mozilla/5.0 (iPhone)", GPUVendor: "Apple Inc.", GPURenderer: "Apple GPU",
		WebGLVersion: "WebGL 2.0", MaxTextureSize: 4096, ContextLost: true, Cores: 6, MemoryGB: 4,
	}, said)
}

func TestAPageThatSaidNothingLeavesTheBrowserAlone(t *testing.T) {
	header := http.Header{}
	header.Set("User-Agent", "Mozilla/5.0")

	assert.Equal(t, notes.Diagnostics{UserAgent: "Mozilla/5.0"}, pagecontext.Diagnostics(nil, header))
	assert.Equal(t, notes.Diagnostics{}, pagecontext.Diagnostics(nil, http.Header{}))
}
