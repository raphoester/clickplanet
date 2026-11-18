import {describe, expect, it} from "vitest"
import {flagNeedOf, LandmassSight, type ScreenView} from "./slotNeed.ts"

const FACING_Z: ScreenView = {
    toward: {x: 0, y: 0, z: 1},
    right: {x: 1, y: 0, z: 0},
    up: {x: 0, y: 1, z: 0},
    origin: {x: 0, y: 0},
    pixelsPerRadian: 1000,
    width: 1000,
    height: 800,
}

// Landmass 1 faces the camera, landmass 2 is behind the earth, landmass 3 is half off screen to the right.
const POSITIONS = new Float32Array([
    0, 0, 1, 0.01, 0, 1, 0, 0.01, 1,
    0, 0, -1,
    0.3, 0, 0.95, 0.7, 0, 0.71,
])
const TILES = [[], [1, 2, 3], [4], [5, 6]]
const sight = new LandmassSight((landmass) => TILES[landmass], TILES.length, POSITIONS, 1e-4)

describe("LandmassSight", () => {
    it("counts the pixels of a landmass on screen, each tile foreshortened by its slant", () => {
        expect(sight.areasOnScreen(FACING_Z)[1]).toBeCloseTo(3 * 1e-4 * 1000 * 1000, 0)
    })

    it("counts nothing of a landmass behind the earth, and only the part on screen", () => {
        expect(sight.areasOnScreen(FACING_Z)[2]).toBe(0)
        expect(sight.areasOnScreen(FACING_Z)[3]).toBeCloseTo(0.95 / Math.hypot(0.3, 0.95) * 1e-4 * 1e6, 0)
    })

    it("stands each sample for its share of a landmass of many tiles", () => {
        const many = Array.from({length: 64}, (_, i) => i + 1)
        const positions = new Float32Array(64 * 3).map((_, i) => (i % 3 === 2 ? 1 : 0))
        const crowd = new LandmassSight((landmass) => (landmass === 1 ? many : []), 2, positions, 1e-4)

        expect(crowd.areasOnScreen(FACING_Z)[1]).toBeCloseTo(64 * 1e-4 * 1e6, 0)
    })
})

describe("flagNeedOf", () => {
    it("is the frame's width on screen, or its height at the flag's aspect, whichever needs more", () => {
        expect(flagNeedOf({x: 0, y: 0, z: 1}, 0.2, 0.1, 1.5, FACING_Z)).toBeCloseTo(400)
        expect(flagNeedOf({x: 0, y: 0, z: 1}, 0.1, 0.1, 1.5, FACING_Z)).toBeCloseTo(300)
    })

    it("foreshortens each axis by its own slant: a frame at the side of the earth needs its height alone", () => {
        expect(flagNeedOf({x: 1, y: 0, z: 0}, 0.2, 0.1, 1.5, FACING_Z)).toBeCloseTo(300)
    })
})
