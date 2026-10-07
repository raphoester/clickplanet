import {describe, expect, it} from "vitest"
import {Point} from "./geometry.ts"
import {SCRIBBLE_BELOW, solidityOf} from "./solidity.ts"

const SPACING = 0.004

// A patch of a hex lattice around the point the camera looks at, one row in two shifted by half a tile.
function tile(column: number, row: number): Point {
    const x = column * SPACING
    const y = (row + (column % 2) / 2) * SPACING
    const length = Math.hypot(x, y, 1)
    return {x: x / length, y: y / length, z: 1 / length}
}

function block(columns: number, rows: number): Point[] {
    return Array.from({length: columns * rows}, (_, i) => tile(i % columns, Math.floor(i / columns)))
}

describe("the solidity of what was taken", () => {
    it("is high for land taken whole", () => {
        const land = block(30, 30)

        expect(solidityOf(land, land)).toBeGreaterThan(0.9)
    })

    it("is high for holes filled in land already held", () => {
        const land = block(30, 30)
        const holes = land.filter((_, i) => i % 7 === 0)

        expect(solidityOf(holes, land)).toBeGreaterThan(0.9)
    })

    it("is low for letters drawn on someone else's land", () => {
        const stroke = Array.from({length: 60}, (_, row) => tile(0, row))
        const bar = Array.from({length: 30}, (_, column) => tile(column, 30))
        const letters = [...stroke, ...bar]

        expect(solidityOf(letters, letters)).toBeLessThan(SCRIBBLE_BELOW)
    })

    it("is whole when nothing taken is held any more", () => {
        expect(solidityOf([], block(5, 5))).toBe(1)
    })
})
