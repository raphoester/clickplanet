import {describe, expect, it} from "vitest"
import {MAX_PICK_WINDOW, pickWindowSize, tilePointSize} from "./pointSize.ts"

describe("tilePointSize", () => {
    it("scales with zoom and viewport height", () => {
        expect(tilePointSize(1, 1000)).toBe(1.5)
        expect(tilePointSize(2, 1000)).toBe(3)
        expect(tilePointSize(1, 2000)).toBe(3)
    })
})

describe("pickWindowSize", () => {
    it("is always odd, so the window has a middle pixel to read", () => {
        for (let size = 0; size <= 200; size += 0.5) {
            expect(pickWindowSize(size) % 2, `pointSize ${size}`).toBe(1)
        }
    })

    it("is wide enough to contain the centre of any sprite covering the middle", () => {
        for (const pointSize of [1.5, 1.9, 9.3, 37.3, 93.2]) {
            const window = pickWindowSize(pointSize)
            const reachFromMiddle = (window - 1) / 2
            expect(reachFromMiddle, `pointSize ${pointSize}`).toBeGreaterThanOrEqual(pointSize / 2)
        }
    })

    it("never goes below the 3 pixels the smallest sprite needs", () => {
        expect(pickWindowSize(0)).toBe(3)
        expect(pickWindowSize(1.5)).toBe(3)
    })

    it("stays bounded, and the bound is odd", () => {
        expect(pickWindowSize(1e6)).toBe(MAX_PICK_WINDOW)
        expect(MAX_PICK_WINDOW % 2).toBe(1)
    })

    it("is not clamped for any viewport a browser will realistically report", () => {
        for (const height of [640, 900, 1243, 2160, 3400]) {
            const pointSize = tilePointSize(50, height)
            expect(pickWindowSize(pointSize), `${height}px tall`).toBeGreaterThanOrEqual(pointSize + 1)
        }
    })

    it("covers the real zoom range, which OrbitControls caps at 50", () => {
        for (const zoom of [1, 5, 10, 25, 50]) {
            const pointSize = tilePointSize(zoom, 1243)
            expect(pickWindowSize(pointSize)).toBeGreaterThanOrEqual(pointSize + 1)
        }
    })
})
