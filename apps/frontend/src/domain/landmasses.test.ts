import {describe, expect, it} from "vitest"
import {Landmasses} from "./landmasses.ts"

describe("Landmasses", () => {
    it("lists the tiles of each landmass, tile ids starting at 1", () => {
        const landmasses = new Landmasses(new Uint16Array([1, 2, 1, 0, 1]), 3)

        expect([...landmasses.tilesOf(1)]).toEqual([1, 3, 5])
        expect([...landmasses.tilesOf(2)]).toEqual([2])
        expect([...landmasses.tilesOf(0)]).toEqual([4])
    })

    it("lists nothing for a landmass the table does not have", () => {
        const landmasses = new Landmasses(new Uint16Array([1, 1]), 2)

        expect([...landmasses.tilesOf(2)]).toEqual([])
        expect([...landmasses.tilesOf(-1)]).toEqual([])
    })
})
