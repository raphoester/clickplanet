import {describe, expect, it} from "vitest"
import {shieldCellsOf} from "./shieldMarks.ts"

describe("shieldCellsOf", () => {
    it("holds the plain shield, every count up to the most and the crown, in as few rows as it can", () => {
        for (const most of [1, 3, 4, 10, 12, 15, 100, 255]) {
            const cells = shieldCellsOf(most)

            expect(cells.x * cells.y).toBeGreaterThanOrEqual(most + 2)
            expect(cells.x * (cells.y - 1)).toBeLessThan(most + 2)
        }
    })

    it("keeps a tile's 10 in four columns, and lays a capital's 100 out square", () => {
        expect(shieldCellsOf(10).toArray()).toEqual([4, 3])
        expect(shieldCellsOf(100).toArray()).toEqual([11, 10])
    })

    it("keeps the biggest a seed can hold under the 4096 pixels every player's GPU takes", () => {
        const cells = shieldCellsOf(255)
        expect(Math.max(cells.x, cells.y) * 64).toBeLessThanOrEqual(4096)
    })
})
