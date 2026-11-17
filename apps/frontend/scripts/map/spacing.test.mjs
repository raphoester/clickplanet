import {describe, expect, it} from "vitest"

import {spacingOf} from "./spacing.mjs"
import {lattice, neighbours} from "./lattice.mjs"

const COLS = 11

const spacing = spacingOf(lattice(COLS).positions, neighbours(COLS))

describe("spacingOf", () => {
    it("is the mean arc between two touching lattice vertices", () => {
        const expected = Math.sqrt(4 * Math.PI / (20 * COLS ** 2) * 4 / Math.sqrt(3))
        expect(spacing).toBeGreaterThan(expected * 0.85)
        expect(spacing).toBeLessThan(expected * 1.15)
    })

    it("halves when the detail doubles", () => {
        const coarse = spacingOf(lattice(8).positions, neighbours(8))
        const fine = spacingOf(lattice(16).positions, neighbours(16))

        expect(fine).toBeGreaterThan(coarse * 0.45)
        expect(fine).toBeLessThan(coarse * 0.55)
    })
})
