package pagecontext

import (
	"net/http"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

// What the page said of itself, and the user agent the server reads off the request.
func Diagnostics(page *supportv1.PageContext, header http.Header) notes.Diagnostics {
	return notes.Diagnostics{
		Build:          page.GetBuild(),
		ViewportWidth:  page.GetViewportWidth(),
		ViewportHeight: page.GetViewportHeight(),
		PixelRatio:     page.GetPixelRatio(),
		ConnectionDown: page.GetConnectionDown(),
		UserAgent:      header.Get("User-Agent"),
		GPUVendor:      page.GetGpuVendor(),
		GPURenderer:    page.GetGpuRenderer(),
		WebGLVersion:   page.GetWebglVersion(),
		MaxTextureSize: page.GetMaxTextureSize(),
		ContextLost:    page.GetContextLost(),
		Cores:          page.GetCores(),
		MemoryGB:       page.GetMemoryGb(),
	}
}
