import {describe, expect, it} from "vitest"
import {ghostTakesOf, Ghosts} from "./ghosts.ts"

describe("Ghosts", () => {
    it("keeps the saved takes that have not run out", () => {
        const ghosts = new Ghosts([
            {tile: 1, flag: "ro", until: 2_000},
            {tile: 2, flag: "ro", until: 1_000},
        ], 1_000)

        expect(ghosts.snapshot()).toEqual([{tile: 1, flag: "ro", until: 2_000}])
    })

    it("paints its flag over the tiles it took", () => {
        const ghosts = new Ghosts([], 0)

        ghosts.take({tile: 7, flag: "tr", until: 5_000})

        expect(ghosts.flagOf(7)).toBe("tr")
        expect(ghosts.flagOf(8)).toBeUndefined()
    })

    it("hands back only the tiles it held when they are released", () => {
        const ghosts = new Ghosts([{tile: 3, flag: "ro", until: 9_000}, {tile: 4, flag: "ro", until: 9_000}], 0)

        expect(ghosts.release([3, 5])).toEqual([3])
        expect(ghosts.snapshot()).toEqual([{tile: 4, flag: "ro", until: 9_000}])
    })

    it("lets each take go at its own time", () => {
        const ghosts = new Ghosts([{tile: 3, flag: "ro", until: 4_000}, {tile: 4, flag: "ro", until: 6_000}], 0)

        expect(ghosts.expire(3_999)).toEqual([])
        expect(ghosts.expire(5_000)).toEqual([3])
        expect(ghosts.expire(6_000)).toEqual([4])
        expect(ghosts.size).toBe(0)
    })

    it("moves each ghost's tile from its holder to the ghost on the board", () => {
        const owners = new Map([[1, "bg"], [2, "bg"], [3, "fr"]])
        const ghosts = new Ghosts([
            {tile: 1, flag: "ro", until: 9_000},
            {tile: 2, flag: "ro", until: 9_000},
            {tile: 3, flag: "ro", until: 9_000},
        ], 0)

        const counts = ghosts.countsOver(new Map([["bg", 10], ["fr", 50]]), (tile) => owners.get(tile))

        expect(counts).toEqual(new Map([["bg", 8], ["fr", 49], ["ro", 3]]))
    })

    it("counts the tiles it holds from a flag", () => {
        const owners = new Map([[1, "bg"], [2, "fr"], [3, "bg"]])
        const ghosts = new Ghosts([
            {tile: 1, flag: "ro", until: 9_000},
            {tile: 2, flag: "ro", until: 9_000},
            {tile: 3, flag: "tr", until: 9_000},
        ], 0)

        expect(ghosts.heldFrom("bg", (tile) => owners.get(tile))).toBe(2)
    })
})

describe("ghostTakesOf", () => {
    it("keeps the well-formed takes on the map and drops the rest", () => {
        const saved = [
            {tile: 1, flag: "ro", until: 9_000},
            {tile: 0, flag: "ro", until: 9_000},
            {tile: 11, flag: "ro", until: 9_000},
            {tile: 2.5, flag: "ro", until: 9_000},
            {tile: 2, flag: "", until: 9_000},
            {tile: 2, flag: "ro", until: "soon"},
            null,
            "tile",
        ]

        expect(ghostTakesOf(saved, 10)).toEqual([{tile: 1, flag: "ro", until: 9_000}])
    })

    it("reads nothing from what is not a list", () => {
        expect(ghostTakesOf({tile: 1}, 10)).toEqual([])
    })
})
