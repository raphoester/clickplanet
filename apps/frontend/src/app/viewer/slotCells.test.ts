import {describe, expect, it} from "vitest"
import {type Cell, CellPacking, cellSize, flagIn, LEVELS, levelFor, levelOf, marginOf, overlap, slotBudgetOf, textureSizeOf, unitsOf} from "./slotCells.ts"

describe("the cells", () => {
    it("halve at each level, a 2:1 cell into squares and a square into 2:1 cells, down to 128 by 64", () => {
        expect(Array.from({length: LEVELS}, (_, level) => cellSize(level))).toEqual([
            {width: 2048, height: 1024}, {width: 1024, height: 1024}, {width: 1024, height: 512}, {width: 512, height: 512},
            {width: 512, height: 256}, {width: 256, height: 256}, {width: 256, height: 128}, {width: 128, height: 128},
            {width: 128, height: 64},
        ])
        for (let level = 0; level < LEVELS; level++) {
            expect(levelOf(cellSize(level))).toBe(level)
            expect(unitsOf(level) * cellSize(LEVELS - 1).width * cellSize(LEVELS - 1).height)
                .toBe(cellSize(level).width * cellSize(level).height)
        }
    })

    it("hold a flag at its aspect inside the margin, centred", () => {
        const cell = {x: 512, y: 256, width: 512, height: 256}
        const flag = flagIn(cell, 1.5)

        expect(marginOf(cell)).toBe(8)
        expect(flag).toEqual({x: 512 + 76, y: 256 + 8, width: 360, height: 240})
        expect(flagIn(cell, 2)).toEqual({x: 528, y: 264, width: 480, height: 240})
    })

    it("are the smallest that holds the flag as wide as it needs, and no larger than the largest allowed", () => {
        expect(levelFor(300, 1.5, 0)).toBe(4)
        expect(flagIn({x: 0, y: 0, ...cellSize(4)}, 1.5).width).toBeGreaterThanOrEqual(300)
        expect(flagIn({x: 0, y: 0, ...cellSize(5)}, 1.5).width).toBeLessThan(300)
        expect(levelFor(5000, 1.5, 1)).toBe(1)
        expect(levelFor(10, 1.5, 0)).toBe(LEVELS - 1)
    })
})

describe("the slot texture", () => {
    it("holds about as many texels as the screen has pixels, and no side past 4096", () => {
        expect(slotBudgetOf(2880, 1800)).toEqual({roots: 2, largest: 0})
        expect(textureSizeOf(slotBudgetOf(2880, 1800))).toEqual({width: 2048, height: 2048})
        expect(slotBudgetOf(786, 1704)).toEqual({roots: 1, largest: 1})
        expect(textureSizeOf(slotBudgetOf(786, 1704))).toEqual({width: 2048, height: 1024})
        expect(textureSizeOf(slotBudgetOf(5120, 2880))).toEqual({width: 2048, height: 4096})
        expect(slotBudgetOf(393, 852)).toEqual({roots: 1, largest: 3})
    })
})

describe("the packing", () => {
    const aligned = ({x, y, width, height}: Cell) => x % width === 0 && y % height === 0

    it("splits a free cell only when none of the size is left, each cell on a multiple of its size", () => {
        const packing = new CellPacking(1)
        const cells = [packing.place(2), packing.place(3), packing.place(3), packing.place(2)]

        expect(cells).toEqual([
            {x: 0, y: 0, width: 1024, height: 512},
            {x: 0, y: 512, width: 512, height: 512},
            {x: 512, y: 512, width: 512, height: 512},
            {x: 1024, y: 0, width: 1024, height: 512},
        ])
        expect(cells.every((cell) => cell && aligned(cell))).toBe(true)
    })

    it("never gives two cells that overlap, and nothing once full", () => {
        const packing = new CellPacking(2)
        const cells: Cell[] = []
        for (const level of [0, 3, 5, 1, 8, 2, 4, 6, 7, 3, 2]) {
            const cell = packing.place(level)
            if (cell) cells.push(cell)
        }
        for (const a of cells) for (const b of cells) if (a !== b) expect(overlap(a, b)).toBe(false)
        expect(cells.every(aligned)).toBe(true)
        expect(cells.every(({y, height}) => y + height <= 2048)).toBe(true)

        const full = new CellPacking(1)
        expect(full.place(0)).toBeDefined()
        expect(full.place(8)).toBeUndefined()
    })

    it("joins two free halves again, so a released texture holds its largest cell", () => {
        const packing = new CellPacking(1)
        const cells = [packing.place(8), packing.place(5), packing.place(3), packing.place(1)]
        expect(packing.place(1)).toBeUndefined()

        for (const cell of cells) packing.release(cell!)

        expect(packing.place(0)).toEqual({x: 0, y: 0, width: 2048, height: 1024})
    })
})
