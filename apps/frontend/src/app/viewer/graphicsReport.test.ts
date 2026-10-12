import {describe, expect, it} from "vitest"
import {GraphicsContext, graphicsReportOf} from "./graphicsReport.ts"

const VENDOR = 0x1F00
const RENDERER = 0x1F01
const VERSION = 0x1F02
const MAX_TEXTURE_SIZE = 0x0D33
const UNMASKED_VENDOR_WEBGL = 0x9245
const UNMASKED_RENDERER_WEBGL = 0x9246

function context(parameters: Record<number, unknown>, options: {unmasked?: boolean, lost?: boolean} = {}): GraphicsContext {
    return {
        VENDOR, RENDERER, VERSION, MAX_TEXTURE_SIZE,
        getParameter: (name: number) => options.lost ? null : parameters[name] ?? null,
        getExtension: (name: string) => name === "WEBGL_debug_renderer_info" && options.unmasked && !options.lost
            ? {UNMASKED_VENDOR_WEBGL, UNMASKED_RENDERER_WEBGL}
            : null,
        isContextLost: () => options.lost === true,
    } as unknown as GraphicsContext
}

const card = {
    [VENDOR]: "WebKit",
    [RENDERER]: "WebKit WebGL",
    [VERSION]: "WebGL 2.0 (OpenGL ES 3.0 Chromium)",
    [MAX_TEXTURE_SIZE]: 16384,
    [UNMASKED_VENDOR_WEBGL]: "Google Inc. (Apple)",
    [UNMASKED_RENDERER_WEBGL]: "ANGLE (Apple, ANGLE Metal Renderer: Apple M1 Pro, Unspecified Version)",
}

describe("graphicsReportOf", () => {
    it("names the card behind the browser when the browser says it", () => {
        expect(graphicsReportOf(context(card, {unmasked: true}))).toEqual({
            vendor: "Google Inc. (Apple)",
            renderer: "ANGLE (Apple, ANGLE Metal Renderer: Apple M1 Pro, Unspecified Version)",
            version: "WebGL 2.0 (OpenGL ES 3.0 Chromium)",
            maxTextureSize: 16384,
            contextLost: false,
        })
    })

    it("gives the browser's own names when it hides the card", () => {
        expect(graphicsReportOf(context(card))).toMatchObject({vendor: "WebKit", renderer: "WebKit WebGL"})
    })

    it("says a lost context is lost, and nothing else", () => {
        expect(graphicsReportOf(context(card, {unmasked: true, lost: true}))).toEqual({
            vendor: "", renderer: "", version: "", maxTextureSize: 0, contextLost: true,
        })
    })
})
