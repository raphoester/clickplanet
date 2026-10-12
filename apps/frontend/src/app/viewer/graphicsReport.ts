export type GraphicsReport = {
    vendor: string
    renderer: string
    version: string
    maxTextureSize: number
    contextLost: boolean
}

export const NO_GRAPHICS: GraphicsReport = {vendor: "", renderer: "", version: "", maxTextureSize: 0, contextLost: false}

export type GraphicsContext = Pick<WebGLRenderingContext,
    "getParameter" | "getExtension" | "isContextLost" | "VENDOR" | "RENDERER" | "VERSION" | "MAX_TEXTURE_SIZE">

// A browser that hides the unmasked names gives its own. A lost context answers null to everything.
export function graphicsReportOf(gl: GraphicsContext): GraphicsReport {
    const unmasked = gl.getExtension("WEBGL_debug_renderer_info")

    return {
        vendor: named(gl.getParameter(unmasked ? unmasked.UNMASKED_VENDOR_WEBGL : gl.VENDOR)),
        renderer: named(gl.getParameter(unmasked ? unmasked.UNMASKED_RENDERER_WEBGL : gl.RENDERER)),
        version: named(gl.getParameter(gl.VERSION)),
        maxTextureSize: counted(gl.getParameter(gl.MAX_TEXTURE_SIZE)),
        contextLost: gl.isContextLost(),
    }
}

function named(value: unknown): string {
    return typeof value === "string" ? value : ""
}

function counted(value: unknown): number {
    return typeof value === "number" && Number.isFinite(value) ? value : 0
}
