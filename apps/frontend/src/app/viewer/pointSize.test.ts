import {describe, expect, it} from "vitest"
import {
    coarseHandover,
    displayPointSize,
    flagPaint,
    MAX_PICK_WINDOW,
    pickWindowSize,
    tilePointSize,
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

    it("covers the real zoom range, from the far view to the cap", () => {
        for (const zoom of [MIN_ZOOM, 1, 5, 10, 25, MAX_ZOOM]) {
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
    it("undoes the widening exactly when the flag stops being painted", () => {
        everyView((zoom, height) => {
            const widened = displayPointSize(zoom, height, "flags") > tilePointSize(zoom, height) + 1e-9
            expect(widened, `zoom ${zoom} at ${height}px`).toBe(flagPaint(zoom, height, "flags") > 0)
        })
    })

    it("covers the ground whenever the flag alone is on screen", () => {
        everyView((zoom, height) => {
            if (flagPaint(zoom, height, "flags") < 1) return
            const spacing = 1.98 * zoom * (height / 1000)
            expect(displayPointSize(zoom, height, "flags") / spacing, `zoom ${zoom} at ${height}px`)
                .toBeGreaterThanOrEqual(1.155)
        })
    })

    it("never draws a disc smaller than the tile it stands for", () => {
        everyView((zoom, height) => {
            expect(displayPointSize(zoom, height, "flags"), `zoom ${zoom} at ${height}px`)
                .toBeGreaterThanOrEqual(tilePointSize(zoom, height))
        })
    })

    it("hands over while a tile is still too small to read a flag in", () => {
        everyView((zoom, height) => {
            const tile = tilePointSize(zoom, height)
            if (tile < 5) expect(flagPaint(zoom, height, "flags"), `zoom ${zoom}`).toBe(1)
            if (tile >= 8) expect(flagPaint(zoom, height, "flags"), `zoom ${zoom}`).toBe(0)
        })
    })

    it("only ever fades one way as you zoom in", () => {
        for (const height of VIEWPORTS) {
            for (let i = 1; i < ZOOMS.length; i++) {
                expect(flagPaint(ZOOMS[i], height, "flags"), `${ZOOMS[i]} at ${height}px`)
                    .toBeLessThanOrEqual(flagPaint(ZOOMS[i - 1], height, "flags"))
            }
        }
    })
})

describe("the map that shows every tile", () => {
    it("never paints a flag over the tiles", () => {
        everyView((zoom, height) => {
            expect(flagPaint(zoom, height, "tiles"), `zoom ${zoom} at ${height}px`).toBe(0)
        })
    })

    it("draws each tile at its own size, at every zoom", () => {
        everyView((zoom, height) => {
            expect(displayPointSize(zoom, height, "tiles"), `zoom ${zoom} at ${height}px`)
                .toBe(tilePointSize(zoom, height))
        })
    })

    it("is the same picture as the flags once they have handed over", () => {
        everyView((zoom, height) => {
            if (flagPaint(zoom, height, "flags") > 0) return
            expect(displayPointSize(zoom, height, "tiles")).toBe(displayPointSize(zoom, height, "flags"))
        })
    })
})
