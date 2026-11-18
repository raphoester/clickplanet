import {describe, expect, it} from "vitest"
import {
    coarseHandover,
    flagPaint,
    flownZoomsOf,
    inOrbit,
    leastFlagRadiusOf,
    tilePointSize,
    tileSpacing,
    tilesZoomOf,
    virusLiftOf,
} from "./pointSize.ts"
import {maxZoomOf, MIN_ZOOM} from "./zoom.ts"
import {BoardScale, EARTH_1_SCALE, scaleOf} from "../../domain/boardScale.ts"
import {FIXTURE_1} from "../../testdata/boards.ts"

const VIEWPORTS = [640, 900, 1243, 2160, 3400]
const STEP = 0.25
const ZOOMS = Array.from(
    {length: Math.round((maxZoomOf(EARTH_1_SCALE) - MIN_ZOOM) / STEP) + 1},
    (_, i) => MIN_ZOOM + i * STEP,
)
const COARSE = scaleOf(FIXTURE_1)

// Each check on earth-1, and on fixture-1 at the zooms that show its tiles as wide.
function everyView(check: (zoom: number, height: number, scale: BoardScale) => void) {
    for (const scale of [EARTH_1_SCALE, COARSE]) {
        for (const height of VIEWPORTS) for (const zoom of ZOOMS) check(zoom / scale.length, height, scale)
    }
}

describe("tilePointSize", () => {
    it("scales with zoom and viewport height", () => {
        expect(tilePointSize(1, 1000, EARTH_1_SCALE)).toBe(1.5)
        expect(tilePointSize(2, 1000, EARTH_1_SCALE)).toBe(3)
        expect(tilePointSize(1, 2000, EARTH_1_SCALE)).toBe(3)
    })

    it("draws a coarser board's tiles as much wider as they are apart", () => {
        expect(tilePointSize(1, 1000, COARSE)).toBeCloseTo(1.5 * COARSE.length, 9)
        expect(tileSpacing(1, 1000, COARSE)).toBeCloseTo(1.98 * COARSE.length, 9)
        expect(tilePointSize(1 / COARSE.length, 1000, COARSE)).toBeCloseTo(1.5, 9)
    })
})

describe("the closest zoom", () => {
    it("is 50 on earth-1", () => {
        expect(maxZoomOf(EARTH_1_SCALE)).toBe(50)
    })

    it("shows a tile as wide on another board", () => {
        expect(maxZoomOf(COARSE)).toBeCloseTo(5.156, 3)
        expect(tilePointSize(maxZoomOf(COARSE), 900, COARSE)).toBeCloseTo(tilePointSize(50, 900, EARTH_1_SCALE), 9)
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
        everyView((zoom, height, scale) => {
            expect(tilePointSize(zoom, height, scale) / tileSpacing(zoom, height, scale), `zoom ${zoom} at ${height}px`).toBeLessThan(0.78)
        })
    })
})

describe("the handover from the painted flag to the tiles", () => {
    it("hands over while a tile is still too small to read a flag in", () => {
        everyView((zoom, height, scale) => {
            const tile = tilePointSize(zoom, height, scale)
            if (tile < 5) expect(flagPaint(zoom, height, "split", scale), `zoom ${zoom}`).toBe(1)
            if (tile >= 8) expect(flagPaint(zoom, height, "split", scale), `zoom ${zoom}`).toBe(0)
        })
    })

    it("only ever fades one way as you zoom in", () => {
        for (const scale of [EARTH_1_SCALE, COARSE]) {
            for (const height of VIEWPORTS) {
                for (let i = 1; i < ZOOMS.length; i++) {
                    expect(flagPaint(ZOOMS[i], height, "split", scale), `${ZOOMS[i]} at ${height}px`)
                        .toBeLessThanOrEqual(flagPaint(ZOOMS[i - 1], height, "split", scale))
                }
            }
        }
    })
})

describe("the zoom the tiles take over from", () => {
    it("is where the painted flag is gone", () => {
        for (const scale of [EARTH_1_SCALE, COARSE]) {
            for (const height of VIEWPORTS) {
                expect(flagPaint(tilesZoomOf(height, scale), height, "split", scale), `${height}px`).toBeCloseTo(0, 9)
                expect(flagPaint(tilesZoomOf(height, scale) * 0.95, height, "split", scale), `${height}px`).toBeGreaterThan(0)
            }
        }
        expect(tilesZoomOf(900, COARSE)).toBeCloseTo(tilesZoomOf(900, EARTH_1_SCALE) / COARSE.length, 9)
    })
})

describe("orbit", () => {
    it("is exactly where the painted flags have the frame to themselves", () => {
        everyView((zoom, height, scale) => {
            expect(inOrbit(zoom, height, scale), `zoom ${zoom} at ${height}px`).toBe(flagPaint(zoom, height, "split", scale) === 1)
        })
    })

    it("is the opening view on a common screen, and ends before the tiles have the frame", () => {
        expect(inOrbit(1, 900, EARTH_1_SCALE)).toBe(true)
        for (const height of VIEWPORTS) expect(inOrbit(tilesZoomOf(height, EARTH_1_SCALE), height, EARTH_1_SCALE), `${height}px`).toBe(false)
    })
})

describe("a territory flown to", () => {
    it("is seen out of orbit, with the tiles alone on screen, and no closer than big tiles", () => {
        for (const scale of [EARTH_1_SCALE, COARSE]) {
            for (const height of VIEWPORTS) {
                const {least, most} = flownZoomsOf(height, scale)
                expect(inOrbit(least, height, scale), `${height}px`).toBe(false)
                expect(flagPaint(least, height, "split", scale), `${height}px`).toBeCloseTo(0, 9)
                expect(tilePointSize(most, height, scale), `${height}px`).toBeCloseTo(28)
                expect(most, `${height}px`).toBeGreaterThan(least)
            }
        }
    })
})

describe("the map that shows every tile", () => {
    it("never paints a flag over the tiles", () => {
        everyView((zoom, height, scale) => {
            expect(flagPaint(zoom, height, "tiles", scale), `zoom ${zoom} at ${height}px`).toBe(0)
        })
    })

})

describe("the shaders' lengths", () => {
    it("are earth-1's own on earth-1", () => {
        expect(virusLiftOf(EARTH_1_SCALE)).toBe(0.012)
        expect(leastFlagRadiusOf(EARTH_1_SCALE)).toBe(0.004)
    })

    it("are as many of a coarser board's spacings", () => {
        expect(virusLiftOf(COARSE)).toBeCloseTo(0.012 * COARSE.length, 12)
        expect(leastFlagRadiusOf(COARSE)).toBeCloseTo(0.004 * COARSE.length, 12)
    })
})
