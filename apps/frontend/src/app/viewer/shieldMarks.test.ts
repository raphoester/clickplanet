import {describe, expect, it} from "vitest"
import {shieldCellsOf} from "./shieldMarks.ts"

describe("shieldCellsOf", () => {
    it("holds the plain shield and every count up to the most, in as few rows as it can", () => {
        for (const most of [1, 3, 4, 10, 12, 15]) {
            const cells = shieldCellsOf(most)

            expect(cells.x * cells.y).toBeGreaterThanOrEqual(most + 1)
            expect(cells.x * (cells.y - 1)).toBeLessThan(most + 1)
        }
    })
})
