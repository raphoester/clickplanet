import {describe, expect, it} from "vitest"
import {outcomeOf, placementOf, TileShields} from "./shields.ts"
import type {Update} from "../backends/backend.ts"

const update = (tile: number, shields: number, newCountry = "fr", previousCountry = newCountry): Update =>
    ({tile, newCountry, previousCountry, clicked: false, shields})

const counts = (shields: TileShields, tiles: number[]) => tiles.map((tile) => shields.shieldsOf(tile))

describe("outcomeOf", () => {
    it("changes nothing on a tile the flag already holds, shielded or not", () => {
        expect(outcomeOf("fr", "fr", 0)).toBe("unchanged")
        expect(outcomeOf("fr", "fr", 4)).toBe("unchanged")
    })

    it("is shielded by the shields standing on another flag's tile, rather than taken", () => {
        expect(outcomeOf("jp", "fr", 1)).toBe("shielded")
    })

    it("takes an unshielded tile, and an empty one", () => {
        expect(outcomeOf("jp", "fr", 0)).toBe("taken")
        expect(outcomeOf(undefined, "fr", 0)).toBe("taken")
    })
})

describe("placementOf", () => {
    it("places a shield on a tile the flag holds", () => {
        expect(placementOf("fr", "fr", 3, 10)).toBe("place")
    })

    it("sends nothing to a tile already at its most", () => {
        expect(placementOf("fr", "fr", 10, 10)).toBe("full")
    })

    it("clicks any tile the flag does not hold", () => {
        expect(placementOf("jp", "fr", 0, 10)).toBe("click")
        expect(placementOf(undefined, "fr", 0, 10)).toBe("click")
    })

    it("leaves the most to the server before the rules say one", () => {
        expect(placementOf("fr", "fr", 40, undefined)).toBe("place")
        expect(placementOf("fr", "fr", 40, 0)).toBe("place")
    })
})

describe("TileShields", () => {
    it("starts every tile with none", () => {
        expect(counts(new TileShields(4), [1, 2, 3, 4])).toEqual([0, 0, 0, 0])
    })

    it("takes the count each update says the tile has now, and reports how it moved", () => {
        const shields = new TileShields(10)

        expect(shields.applyUpdates([update(1, 3), update(2, 1)])).toEqual([
            {tile: 1, shields: 3, was: 0},
            {tile: 2, shields: 1, was: 0},
        ])
        expect(shields.applyUpdates([update(1, 2), update(2, 0, "jp", "fr")])).toEqual([
            {tile: 1, shields: 2, was: 3},
            {tile: 2, shields: 0, was: 1},
        ])
        expect(counts(shields, [1, 2])).toEqual([2, 0])
    })

    it("reports nothing for an update that leaves the count where it was", () => {
        const shields = new TileShields(10)
        shields.applyUpdates([update(1, 2)])

        expect(shields.applyUpdates([update(1, 2), update(3, 0)])).toEqual([])
    })

    it("seeds the counts a map batch lists", () => {
        const shields = new TileShields(10)

        expect(shields.applyBatch(new Map([[4, 7], [9, 1]]))).toEqual([
            {tile: 4, shields: 7, was: 0},
            {tile: 9, shields: 1, was: 0},
        ])
        expect(counts(shields, [4, 5, 9])).toEqual([7, 0, 1])
    })

    it("lets a live update win over a batch that was already in flight", () => {
        const shields = new TileShields(10)
        shields.applyUpdates([update(4, 2)])
        shields.applyUpdates([update(5, 0, "jp", "fr")])

        expect(shields.applyBatch(new Map([[4, 7], [5, 3], [6, 1]]))).toEqual([{tile: 6, shields: 1, was: 0}])
        expect(counts(shields, [4, 5, 6])).toEqual([2, 0, 1])
    })

    it("empties a cleared tile, and keeps the batch off it", () => {
        const shields = new TileShields(10)
        shields.applyBatch(new Map([[2, 3]]))

        expect(shields.applyClears([2, 3])).toEqual([{tile: 2, shields: 0, was: 3}])
        expect(shields.applyBatch(new Map([[2, 5]]))).toEqual([])
    })

    it("takes one shield off each struck tile, never under none", () => {
        const shields = new TileShields(10)
        shields.applyUpdates([update(1, 3), update(2, 1)])

        expect(shields.applyStrikes([1, 2, 3])).toEqual([
            {tile: 1, shields: 2, was: 3},
            {tile: 2, shields: 0, was: 1},
        ])
        expect(counts(shields, [1, 2, 3])).toEqual([2, 0, 0])
    })

    it("ignores tiles outside the map", () => {
        const shields = new TileShields(4)

        expect(shields.applyUpdates([update(0, 2), update(5, 2)])).toEqual([])
        expect(shields.applyBatch(new Map([[9, 1]]))).toEqual([])
        expect(shields.applyStrikes([-1])).toEqual([])
        expect(shields.shieldsOf(5)).toBe(0)
    })
})
