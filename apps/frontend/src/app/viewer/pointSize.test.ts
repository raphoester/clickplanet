import {describe, expect, it} from "vitest"
import {
    coarseHandover,
    flagPaint,
    flownZoomsOf,
    inOrbit,
    tilePointSize,
    tileSpacing,
    tilesZoomOf,
} from "./pointSize.ts"
import {MAX_ZOOM, MIN_ZOOM} from "./zoom.ts"

const VIEWPORTS = [640, 900, 1243, 2160, 3400]
const STEP = 0.25
const ZOOMS = Array.from(
    {length: Math.round((MAX_ZOOM - MIN_ZOOM) / STEP) + 1},
    (_, i) => MIN_ZOOM + i * STEP,
)

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

describe("a tile's disc", () => {
    // Two neighbours are 0.78 of the median spacing apart at the lattice's 1st percentile, 0.73 at its tightest.
    it("stays off its neighbours' but round the lattice's 12 corners", () => {
        everyView((zoom, height) => {
            expect(tilePointSize(zoom, height) / tileSpacing(zoom, height), `zoom ${zoom} at ${height}px`).toBeLessThan(0.78)
        })
    })
})

describe("the handover from the painted flag to the tiles", () => {
    it("hands over while a tile is still too small to read a flag in", () => {
        everyView((zoom, height) => {
            const tile = tilePointSize(zoom, height)
            if (tile < 5) expect(flagPaint(zoom, height, "split"), `zoom ${zoom}`).toBe(1)
            if (tile >= 8) expect(flagPaint(zoom, height, "split"), `zoom ${zoom}`).toBe(0)
        })
    })

    it("only ever fades one way as you zoom in", () => {
        for (const height of VIEWPORTS) {
            for (let i = 1; i < ZOOMS.length; i++) {
                expect(flagPaint(ZOOMS[i], height, "split"), `${ZOOMS[i]} at ${height}px`)
                    .toBeLessThanOrEqual(flagPaint(ZOOMS[i - 1], height, "split"))
            }
        }
    })
})

describe("the zoom the tiles take over from", () => {
    it("is where the painted flag is gone", () => {
        for (const height of VIEWPORTS) {
            expect(flagPaint(tilesZoomOf(height), height, "split"), `${height}px`).toBeCloseTo(0, 9)
            expect(flagPaint(tilesZoomOf(height) * 0.95, height, "split"), `${height}px`).toBeGreaterThan(0)
        }
    })
})

describe("orbit", () => {
    it("is exactly where the painted flags have the frame to themselves", () => {
        everyView((zoom, height) => {
            expect(inOrbit(zoom, height), `zoom ${zoom} at ${height}px`).toBe(flagPaint(zoom, height, "split") === 1)
        })
    })

    it("is the opening view on a common screen, and ends before the tiles have the frame", () => {
        expect(inOrbit(1, 900)).toBe(true)
        for (const height of VIEWPORTS) expect(inOrbit(tilesZoomOf(height), height), `${height}px`).toBe(false)
    })
})

describe("a territory flown to", () => {
    it("is seen out of orbit, with the tiles alone on screen, and no closer than big tiles", () => {
        for (const height of VIEWPORTS) {
            const {least, most} = flownZoomsOf(height)
            expect(inOrbit(least, height), `${height}px`).toBe(false)
            expect(flagPaint(least, height, "split"), `${height}px`).toBeCloseTo(0, 9)
            expect(tilePointSize(most, height), `${height}px`).toBeCloseTo(28)
            expect(most, `${height}px`).toBeGreaterThan(least)
        }
    })
})

describe("the map that shows every tile", () => {
    it("never paints a flag over the tiles", () => {
        everyView((zoom, height) => {
            expect(flagPaint(zoom, height, "tiles"), `zoom ${zoom} at ${height}px`).toBe(0)
        })
    })

})
