import {describe, expect, it} from "vitest"
import {outcomeOf, placementOf, TileGarrisons} from "./garrisons.ts"
import type {Update} from "../backends/backend.ts"

const update = (tile: number, defenders: number, newCountry = "fr", previousCountry = newCountry): Update =>
    ({tile, newCountry, previousCountry, clicked: false, defenders})

const counts = (garrisons: TileGarrisons, tiles: number[]) => tiles.map((tile) => garrisons.defendersOf(tile))

describe("outcomeOf", () => {
    it("changes nothing on a tile the flag already holds, defended or not", () => {
        expect(outcomeOf("fr", "fr", 0)).toBe("unchanged")
        expect(outcomeOf("fr", "fr", 4)).toBe("unchanged")
    })

    it("is defended by the defenders standing on another flag's tile, rather than taken", () => {
        expect(outcomeOf("jp", "fr", 1)).toBe("defended")
    })

    it("takes an undefended tile, and an empty one", () => {
        expect(outcomeOf("jp", "fr", 0)).toBe("taken")
        expect(outcomeOf(undefined, "fr", 0)).toBe("taken")
    })
})

describe("placementOf", () => {
    it("places a defender on a tile the flag holds", () => {
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

describe("TileGarrisons", () => {
    it("starts every tile with none", () => {
        expect(counts(new TileGarrisons(4), [1, 2, 3, 4])).toEqual([0, 0, 0, 0])
    })

    it("takes the count each update says the tile has now, and reports how it moved", () => {
        const garrisons = new TileGarrisons(10)

        expect(garrisons.applyUpdates([update(1, 3), update(2, 1)])).toEqual([
            {tile: 1, defenders: 3, was: 0},
            {tile: 2, defenders: 1, was: 0},
        ])
        expect(garrisons.applyUpdates([update(1, 2), update(2, 0, "jp", "fr")])).toEqual([
            {tile: 1, defenders: 2, was: 3},
            {tile: 2, defenders: 0, was: 1},
        ])
        expect(counts(garrisons, [1, 2])).toEqual([2, 0])
    })

    it("reports nothing for an update that leaves the count where it was", () => {
        const garrisons = new TileGarrisons(10)
        garrisons.applyUpdates([update(1, 2)])

        expect(garrisons.applyUpdates([update(1, 2), update(3, 0)])).toEqual([])
    })

    it("seeds the counts a map batch lists", () => {
        const garrisons = new TileGarrisons(10)

        expect(garrisons.applyBatch(new Map([[4, 7], [9, 1]]))).toEqual([
            {tile: 4, defenders: 7, was: 0},
            {tile: 9, defenders: 1, was: 0},
        ])
        expect(counts(garrisons, [4, 5, 9])).toEqual([7, 0, 1])
    })

    it("lets a live update win over a batch that was already in flight", () => {
        const garrisons = new TileGarrisons(10)
        garrisons.applyUpdates([update(4, 2)])
        garrisons.applyUpdates([update(5, 0, "jp", "fr")])

        expect(garrisons.applyBatch(new Map([[4, 7], [5, 3], [6, 1]]))).toEqual([{tile: 6, defenders: 1, was: 0}])
        expect(counts(garrisons, [4, 5, 6])).toEqual([2, 0, 1])
    })

    it("empties a cleared tile, and keeps the batch off it", () => {
        const garrisons = new TileGarrisons(10)
        garrisons.applyBatch(new Map([[2, 3]]))

        expect(garrisons.applyClears([2, 3])).toEqual([{tile: 2, defenders: 0, was: 3}])
        expect(garrisons.applyBatch(new Map([[2, 5]]))).toEqual([])
    })

    it("takes one defender off each struck tile, never under none", () => {
        const garrisons = new TileGarrisons(10)
        garrisons.applyUpdates([update(1, 3), update(2, 1)])

        expect(garrisons.applyStrikes([1, 2, 3])).toEqual([
            {tile: 1, defenders: 2, was: 3},
            {tile: 2, defenders: 0, was: 1},
        ])
        expect(counts(garrisons, [1, 2, 3])).toEqual([2, 0, 0])
    })

    it("ignores tiles outside the map", () => {
        const garrisons = new TileGarrisons(4)

        expect(garrisons.applyUpdates([update(0, 2), update(5, 2)])).toEqual([])
        expect(garrisons.applyBatch(new Map([[9, 1]]))).toEqual([])
        expect(garrisons.applyStrikes([-1])).toEqual([])
        expect(garrisons.defendersOf(5)).toBe(0)
    })
})
