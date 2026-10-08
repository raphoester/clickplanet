import {describe, expect, it} from "vitest"
import {foughtOver, losersOf, takersFrom, TileChange} from "./changes.ts"

function took(to: string | undefined, from: string | undefined, count: number, first = 1, at = 0): TileChange[] {
    return Array.from({length: count}, (_, i) => ({tile: first + i, from, to, at}))
}

describe("the countries fought over", () => {
    const ground = (tile: number) => tile <= 1000 ? "de" : tile <= 1500 ? "pl" : "fr"

    it("are the ones whose ground changed hands enough, the most first", () => {
        const changes = [...took("ro", "de", 600, 1), ...took("ro", "pl", 450, 1001), ...took("it", "fr", 300, 1501)]

        expect(foughtOver(changes, ground, 0, 10)).toEqual(["de", "pl"])
    })

    it("count only land taken from another flag, in the window", () => {
        const changes = [...took("ro", undefined, 600, 1), ...took("ro", "de", 600, 1, 50)]

        expect(foughtOver(changes, ground, 0, 10)).toEqual([])
    })
})

describe("the losers and the takers", () => {
    const changes = [...took("il", "dz", 30), ...took("es", "dz", 20, 31), ...took("bg", "ps", 10, 51), ...took(undefined, "fr", 40, 61)]

    it("are ranked by tiles lost to another flag, a bomb's crater aside", () => {
        expect(losersOf(changes)).toEqual(["dz", "ps"])
    })

    it("take from one loser, the most first", () => {
        expect(takersFrom(changes, "dz")).toEqual(["il", "es"])
    })
})
