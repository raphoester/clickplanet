// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {layoutViewport} from "./viewport.ts"

function rootReports(width: number, height: number) {
    Object.defineProperty(document.documentElement, "clientWidth", {value: width, configurable: true})
    Object.defineProperty(document.documentElement, "clientHeight", {value: height, configurable: true})
}

afterEach(() => rootReports(0, 0))

describe("layoutViewport", () => {
    // `window.innerWidth` follows the pinch on iOS Safari, so believing it
    // sizes the canvas to the zoomed portion and leaves black beside the globe
    // once the zoom is released.
    it("reads the root's client box, not the zoom-following window", () => {
        rootReports(390, 844)
        window.innerWidth = 341
        window.innerHeight = 738

        expect(layoutViewport()).toEqual({width: 390, height: 844})
    })

    it("falls back to the window where the root reports no box", () => {
        rootReports(0, 0)
        window.innerWidth = 1024
        window.innerHeight = 768

        expect(layoutViewport()).toEqual({width: 1024, height: 768})
    })
})
