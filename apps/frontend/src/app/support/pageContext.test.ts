import {describe, expect, it} from "vitest"
import {BUILD, buildOf, pageContextOf} from "./pageContext.ts"
import {NO_GRAPHICS} from "../viewer/graphicsReport.ts"

describe("buildOf", () => {
    it("is the file a module runs from, which carries the build's hash", () => {
        expect(buildOf("https://clickplanet.lol/assets/play-a1B2c3D4.js")).toBe("play-a1B2c3D4.js")
        expect(buildOf("http://localhost:5173/src/app/support/pageContext.ts?t=1")).toBe("pageContext.ts")
    })

    it("is empty for what is not a URL", () => {
        expect(buildOf("not a url")).toBe("")
    })
})

describe("pageContextOf", () => {
    const phone = {
        innerWidth: 390.4,
        innerHeight: 843.6,
        devicePixelRatio: 3,
        navigator: {hardwareConcurrency: 6, deviceMemory: 4},
    }
    const card = {
        vendor: "Apple Inc.",
        renderer: "Apple GPU",
        version: "WebGL 2.0",
        maxTextureSize: 4096,
        contextLost: true,
    }

    it("says the build, the page in whole pixels, the device, the card and whether the server was lost", () => {
        expect(pageContextOf(phone, card, true)).toEqual({
            build: BUILD,
            viewportWidth: 390,
            viewportHeight: 844,
            pixelRatio: 3,
            connectionDown: true,
            gpuVendor: "Apple Inc.",
            gpuRenderer: "Apple GPU",
            webglVersion: "WebGL 2.0",
            maxTextureSize: 4096,
            contextLost: true,
            cores: 6,
            memoryGb: 4,
        })
        expect(BUILD).toBe("pageContext.ts")
    })

    it("says nothing of what the browser does not tell", () => {
        expect(pageContextOf({innerWidth: 1280, innerHeight: 720, navigator: {}}, NO_GRAPHICS, false)).toEqual({
            build: BUILD,
            viewportWidth: 1280,
            viewportHeight: 720,
            pixelRatio: 0,
            connectionDown: false,
            gpuVendor: "",
            gpuRenderer: "",
            webglVersion: "",
            maxTextureSize: 0,
            contextLost: false,
            cores: 0,
            memoryGb: 0,
        })
    })
})
