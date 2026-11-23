import {describe, expect, it} from "vitest"
import {heightFor, MOST_HEIGHT, soldCellOf} from "./flagAtlas.ts"
import {FIRST_SOLD, regions, SOLD_MOST, SPRITE_WIDTH} from "./atlas.ts"
import {ATLAS_SIZE} from "./atlasAsset.ts"

const SHAPE = {width: 768, height: 512}

describe("a sold flag's cell", () => {
    it("is under the built atlas, a sprite wide and as high as its picture is at that width", () => {
        const cell = soldCellOf(FIRST_SOLD, SHAPE)
        expect(cell).toEqual({x: 0, y: ATLAS_SIZE.height + 16, width: SPRITE_WIDTH, height: 67})
        expect(soldCellOf(FIRST_SOLD, {width: 512, height: 512}).height).toBe(100)
        expect(soldCellOf(FIRST_SOLD, {width: 768, height: 384}).height).toBe(50)
    })

    it("fills a row across the atlas's width, then the next", () => {
        const perRow = ATLAS_SIZE.width / SPRITE_WIDTH
        expect(soldCellOf(FIRST_SOLD + 1, SHAPE).x).toBe(SPRITE_WIDTH)
        expect(soldCellOf(FIRST_SOLD + perRow, SHAPE)).toMatchObject({x: 0, y: ATLAS_SIZE.height + 16 + 100})
    })

    it("never meets a built flag nor another cell, and every one fits a texture a phone takes", () => {
        const cells = Array.from({length: SOLD_MOST}, (_, i) => soldCellOf(FIRST_SOLD + i, {width: 512, height: 512}))
        const meets = (a: {x: number, y: number, width: number, height: number}, b: typeof a) =>
            a.x < b.x + b.width && b.x < a.x + a.width && a.y < b.y + b.height && b.y < a.y + a.height
        expect(cells.filter((cell) => [...regions.values()].some((built) => meets(cell, built)))).toEqual([])
        expect(cells.filter((cell, i) => cells.some((other, j) => i !== j && meets(cell, other)))).toEqual([])
        expect(heightFor(Math.max(...cells.map((cell) => cell.y + 100)))).toBeLessThanOrEqual(MOST_HEIGHT)
    })
})

describe("the texture's height", () => {
    it("grows under the built atlas by whole steps, which keep its place in the mip levels", () => {
        expect(heightFor(ATLAS_SIZE.height)).toBe(ATLAS_SIZE.height)
        expect(heightFor(ATLAS_SIZE.height + 1)).toBe(ATLAS_SIZE.height + 256)
        expect(heightFor(ATLAS_SIZE.height + 256)).toBe(ATLAS_SIZE.height + 256)
        expect(heightFor(ATLAS_SIZE.height + 257)).toBe(ATLAS_SIZE.height + 512)
    })
})
