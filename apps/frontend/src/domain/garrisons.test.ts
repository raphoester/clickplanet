import {describe, expect, it} from "vitest"
import {outcomeOf, placementOf, TileGarrisons} from "./garrisons.ts"
import {TileOwnership} from "./tileOwnership.ts"

const held = (owners: Record<number, string>) => {
    const ownership = new TileOwnership(10)
    ownership.applyBatch({bindings: new Map(Object.entries(owners).map(([tile, country]) => [Number(tile), country]))})
    return ownership
}

const loaded = (ownership: TileOwnership) => {
    const garrisons = new TileGarrisons((tile) => ownership.ownerOf(tile))
    garrisons.applyLoaded([])
    return garrisons
}

describe("outcomeOf", () => {
    it("changes nothing on a tile the flag already holds, defended or not", () => {
        expect(outcomeOf("fr", "fr", 0)).toBe("unchanged")
        expect(outcomeOf("fr", "fr", 4)).toBe("unchanged")
    })

    it("is defended by a garrison standing on another flag's tile, rather than taken", () => {
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
    it("keeps the count each garrison sends, and reports how it moved", () => {
        const garrisons = loaded(held({1: "fr"}))

        expect(garrisons.apply({tile: 1, country: "fr", defenders: 3})).toEqual([{tile: 1, defenders: 3, was: 0}])
        expect(garrisons.apply({tile: 1, country: "fr", defenders: 2})).toEqual([{tile: 1, defenders: 2, was: 3}])
        expect(garrisons.defendersOf(1)).toBe(2)
    })

    it("removes a garrison with no defender left", () => {
        const garrisons = loaded(held({1: "fr"}))
        garrisons.apply({tile: 1, country: "fr", defenders: 1})

        expect(garrisons.apply({tile: 1, country: "fr", defenders: 0})).toEqual([{tile: 1, defenders: 0, was: 1}])
        expect(garrisons.defendersOf(1)).toBe(0)
    })

    it("reports nothing when the count did not move", () => {
        const garrisons = loaded(held({1: "fr"}))
        garrisons.apply({tile: 1, country: "fr", defenders: 2})

        expect(garrisons.apply({tile: 1, country: "fr", defenders: 2})).toEqual([])
        expect(garrisons.apply({tile: 2, country: "fr", defenders: 0})).toEqual([])
    })

    it("ignores a garrison for a flag the tile does not wear", () => {
        const garrisons = loaded(held({1: "fr"}))

        expect(garrisons.apply({tile: 1, country: "jp", defenders: 3})).toEqual([])
        expect(garrisons.apply({tile: 2, country: "jp", defenders: 3})).toEqual([])
        expect(garrisons.defendersOf(1)).toBe(0)
    })

    it("drops a garrison when its tile changes hands, and keeps it when the flag stays", () => {
        const garrisons = loaded(held({1: "fr", 2: "fr", 3: "fr"}))
        garrisons.apply({tile: 1, country: "fr", defenders: 4})
        garrisons.apply({tile: 2, country: "fr", defenders: 2})
        garrisons.apply({tile: 3, country: "fr", defenders: 1})

        expect(garrisons.followOwners([
            {tile: 1, country: "jp"},
            {tile: 2, country: undefined},
            {tile: 3, country: "fr"},
            {tile: 4, country: "jp"},
        ])).toEqual([
            {tile: 1, defenders: 0, was: 4},
            {tile: 2, defenders: 0, was: 2},
        ])
        expect([1, 2, 3].map((tile) => garrisons.defendersOf(tile))).toEqual([0, 0, 1])
    })

    it("does not bring a dropped garrison back when its flag retakes the tile", () => {
        const ownership = held({1: "fr"})
        const garrisons = loaded(ownership)
        garrisons.apply({tile: 1, country: "fr", defenders: 4})

        garrisons.followOwners(ownership.applyUpdates([{tile: 1, previousCountry: "fr", newCountry: "jp", clicked: true}]))
        garrisons.followOwners(ownership.applyUpdates([{tile: 1, previousCountry: "jp", newCountry: "fr", clicked: true}]))

        expect(garrisons.defendersOf(1)).toBe(0)
    })

    describe("while the map loads", () => {
        it("holds what the stream says until the map and the read are in, then lets it win", () => {
            const ownership = new TileOwnership(10)
            const garrisons = new TileGarrisons((tile) => ownership.ownerOf(tile))

            expect(garrisons.apply({tile: 1, country: "fr", defenders: 5})).toEqual([])
            expect(garrisons.defendersOf(1)).toBe(0)

            ownership.applyBatch({bindings: new Map([[1, "fr"], [2, "jp"]])})
            expect(garrisons.applyLoaded([
                {tile: 1, country: "fr", defenders: 3},
                {tile: 2, country: "jp", defenders: 7},
            ])).toEqual([
                {tile: 1, defenders: 3, was: 0},
                {tile: 2, defenders: 7, was: 0},
                {tile: 1, defenders: 5, was: 3},
            ])
            expect(garrisons.defendersOf(1)).toBe(5)
            expect(garrisons.defendersOf(2)).toBe(7)
        })

        it("leaves out a garrison read for a flag the loaded tile no longer wears", () => {
            const garrisons = new TileGarrisons((tile) => ({1: "jp"} as Record<number, string>)[tile])

            expect(garrisons.applyLoaded([{tile: 1, country: "fr", defenders: 3}])).toEqual([])
            expect(garrisons.defendersOf(1)).toBe(0)
        })

        it("applies the stream at once after the load", () => {
            const garrisons = loaded(held({1: "fr"}))

            expect(garrisons.apply({tile: 1, country: "fr", defenders: 1})).toEqual([{tile: 1, defenders: 1, was: 0}])
        })
    })
})
