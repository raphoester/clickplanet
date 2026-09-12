import {describe, expect, it} from "vitest"
import {offScreen, placePointer} from "./bonusPointer.ts"

describe("offScreen", () => {
    it("says nothing when the box is in the frame", () => {
        expect(offScreen({x: 0, y: 0})).toBe(false)
        expect(offScreen({x: 0.99, y: -0.99})).toBe(false)
    })

    it("catches a box past either edge", () => {
        expect(offScreen({x: 1.4, y: 0})).toBe(true)
        expect(offScreen({x: 0, y: -3})).toBe(true)
    })

    it("catches one past a corner", () => {
        expect(offScreen({x: 2, y: 2})).toBe(true)
    })
})

describe("placePointer", () => {
    const margin = 0.1

    it("puts the pointer on the inset edge it runs out of room on first", () => {
        const {x, y} = placePointer({x: 4, y: 1}, margin)

        expect(x).toBeCloseTo(0.9, 6)
        expect(y).toBeCloseTo(0.225, 6)
    })

    it("uses the other edge when that one is nearer", () => {
        const {x, y} = placePointer({x: 1, y: 4}, margin)

        expect(y).toBeCloseTo(0.9, 6)
        expect(x).toBeCloseTo(0.225, 6)
    })

    it("stays inside the frame whichever way the box went", () => {
        for (let step = 0; step < 360; step++) {
            const angle = (step / 360) * 2 * Math.PI
            const {x, y} = placePointer({x: Math.cos(angle) * 9, y: Math.sin(angle) * 9}, margin)

            expect(Math.abs(x)).toBeLessThanOrEqual(0.9 + 1e-9)
            expect(Math.abs(y)).toBeLessThanOrEqual(0.9 + 1e-9)
        }
    })

    it("keeps the pointer on the ray to the box, so it does not lie about the way", () => {
        const box = {x: 3, y: -1.5}
        const {x, y} = placePointer(box, margin)

        expect(Math.atan2(y, x)).toBeCloseTo(Math.atan2(box.y, box.x), 6)
    })

    it("turns to face the box, in screen degrees", () => {
        // Screen y grows downward, so a box above the middle is a negative angle.
        expect(placePointer({x: 5, y: 0}, margin).angle).toBeCloseTo(0, 6)
        expect(placePointer({x: 0, y: 5}, margin).angle).toBeCloseTo(-90, 6)
        expect(placePointer({x: 0, y: -5}, margin).angle).toBeCloseTo(90, 6)

        // Straight left is a half turn either way round.
        expect(Math.abs(placePointer({x: -5, y: 0}, margin).angle)).toBeCloseTo(180, 6)
    })

    it("does not divide by a zero direction", () => {
        const {x, y, angle} = placePointer({x: 0, y: 0}, margin)

        expect(Number.isFinite(x)).toBe(true)
        expect(Number.isFinite(y)).toBe(true)
        expect(Number.isFinite(angle)).toBe(true)
    })
})
