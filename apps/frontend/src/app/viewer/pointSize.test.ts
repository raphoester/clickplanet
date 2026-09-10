import {describe, expect, it} from "vitest"
import {
    coarseHandover,
    displayPointSize,
    flagPaint,
    MAX_PICK_WINDOW,
    pickWindowSize,
    tilePointSize,
} from "./pointSize.ts"

// Every zoom OrbitControls allows, on viewports a browser really reports.
const VIEWPORTS = [640, 900, 1243, 2160, 3400]
const ZOOMS = Array.from({length: 197}, (_, i) => 1 + i * 0.25)   // 1 to 50

function everyView(check: (zoom: number, height: number) => void) {
    for (const height of VIEWPORTS) for (const zoom of ZOOMS) check(zoom, height)
}

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

describe("coarseHandover", () => {
    it("is a fraction, whatever it is handed", () => {
        for (const size of [-10, 0, 4.9, 5, 6.5, 8, 8.1, 1e6]) {
            expect(coarseHandover(size), `size ${size}`).toBeGreaterThanOrEqual(0)
            expect(coarseHandover(size), `size ${size}`).toBeLessThanOrEqual(1)
        }
    })

    it("runs from nothing to everything across the handover", () => {
        expect(coarseHandover(5)).toBe(0)
        expect(coarseHandover(6.5)).toBeCloseTo(0.5)
        expect(coarseHandover(8)).toBe(1)
    })
})

describe("the handover from the painted flag to the tiles", () => {
    // The widening exists to give the painted flag a surface to land on. If it
    // outlasts the flag, tiles you are about to aim at are fattened for nothing;
    // if it ends first, the flag is painted through a lattice full of holes and
    // loses ink to them. Neither is visible in a screenshot of one zoom, which
    // is why it is pinned here.
    it("undoes the widening exactly when the flag stops being painted", () => {
        everyView((zoom, height) => {
            const widened = displayPointSize(zoom, height) > tilePointSize(zoom, height) + 1e-9
            expect(widened, `zoom ${zoom} at ${height}px`).toBe(flagPaint(zoom, height) > 0)
        })
    })

    it("covers the ground whenever the flag alone is on screen", () => {
        // Circles on this hex lattice cover it fully at 1.155x the spacing, and
        // the tiles sit ~1.98px apart at zoom 1 on a 1000px globe.
        everyView((zoom, height) => {
            if (flagPaint(zoom, height) < 1) return
            const spacing = 1.98 * zoom * (height / 1000)
            expect(displayPointSize(zoom, height) / spacing, `zoom ${zoom} at ${height}px`)
                .toBeGreaterThanOrEqual(1.155)
        })
    })

    it("never draws a disc smaller than the tile it stands for", () => {
        everyView((zoom, height) => {
            expect(displayPointSize(zoom, height), `zoom ${zoom} at ${height}px`)
                .toBeGreaterThanOrEqual(tilePointSize(zoom, height))
        })
    })

    it("hands over while a tile is still too small to read a flag in", () => {
        // Below ~8px a 100px sprite in a disc is noise either way, mip-blurred
        // to grey or aliased into sparkle. The flag has to be gone by then, and
        // must still be there while the tiles are smaller than that.
        everyView((zoom, height) => {
            const tile = tilePointSize(zoom, height)
            if (tile < 5) expect(flagPaint(zoom, height), `zoom ${zoom}`).toBe(1)
            if (tile >= 8) expect(flagPaint(zoom, height), `zoom ${zoom}`).toBe(0)
        })
    })

    it("only ever fades one way as you zoom in", () => {
        for (const height of VIEWPORTS) {
            for (let i = 1; i < ZOOMS.length; i++) {
                expect(flagPaint(ZOOMS[i], height), `${ZOOMS[i]} at ${height}px`)
                    .toBeLessThanOrEqual(flagPaint(ZOOMS[i - 1], height))
            }
        }
    })
})
