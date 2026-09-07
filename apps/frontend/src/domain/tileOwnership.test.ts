import {describe, expect, it} from "vitest"
import {TileOwnership} from "./tileOwnership.ts"
import type {Update} from "../backends/backend.ts"

const batch = (bindings: Record<number, string>) => ({
    bindings: new Map(Object.entries(bindings).map(([k, v]) => [Number(k), v])),
})

const update = (tile: number, newCountry: string, previousCountry?: string): Update =>
    ({tile, newCountry, previousCountry})

const counts = (store: TileOwnership) => Object.fromEntries(store.counts())

describe("applyBatch", () => {
    it("assigns owners and reports what changed", () => {
        const store = new TileOwnership(10)
        expect(store.applyBatch(batch({1: "fr", 2: "jp"}))).toEqual([
            {tile: 1, country: "fr"},
            {tile: 2, country: "jp"},
        ])
        expect(store.ownerOf(1)).toBe("fr")
        expect(counts(store)).toEqual({fr: 1, jp: 1})
    })

    it("reports nothing for a tile whose owner did not change", () => {
        const store = new TileOwnership(10)
        store.applyBatch(batch({1: "fr"}))
        expect(store.applyBatch(batch({1: "fr"}))).toEqual([])
        expect(counts(store)).toEqual({fr: 1})
    })

    it("moves the count across when a batch reassigns a tile", () => {
        const store = new TileOwnership(10)
        store.applyBatch(batch({1: "fr"}))
        store.applyBatch(batch({1: "jp"}))
        expect(counts(store)).toEqual({jp: 1})
    })
})

describe("applyUpdates", () => {
    it("assigns owners and reports what changed", () => {
        const store = new TileOwnership(10)
        expect(store.applyUpdates([update(3, "de")])).toEqual([{tile: 3, country: "de"}])
        expect(counts(store)).toEqual({de: 1})
    })

    it("moves the count from the previous owner", () => {
        const store = new TileOwnership(10)
        store.applyUpdates([update(1, "fr")])
        store.applyUpdates([update(1, "jp", "fr")])
        expect(counts(store)).toEqual({jp: 1})
    })

    /**
     * The previous owner reported by the server is not trusted: counts come
     * from the map we hold. Trusting the event is what let a count drift, and
     * eventually go negative, whenever the two disagreed.
     */
    it("ignores a previous owner that disagrees with what it holds", () => {
        const store = new TileOwnership(10)
        store.applyUpdates([update(1, "fr")])
        store.applyUpdates([update(1, "jp", "de")]) // server thinks it was German
        expect(counts(store)).toEqual({jp: 1})
    })

    it("never lets a count go negative", () => {
        const store = new TileOwnership(10)
        store.applyUpdates([update(1, "jp", "fr")]) // France never owned anything
        expect(counts(store)).toEqual({jp: 1})
        expect(store.counts().has("fr")).toBe(false)
    })

    it("drops a country from the counts once it holds no tiles", () => {
        const store = new TileOwnership(10)
        store.applyUpdates([update(1, "fr")])
        store.applyUpdates([update(1, "jp", "fr")])
        expect(store.counts().has("fr")).toBe(false)
    })

    it("reports nothing when an update repeats the owner it already has", () => {
        const store = new TileOwnership(10)
        store.applyUpdates([update(1, "fr")])
        expect(store.applyUpdates([update(1, "fr")])).toEqual([])
        expect(counts(store)).toEqual({fr: 1})
    })
})

/**
 * The initial load takes ~26 sequential requests over several seconds, and
 * live updates stream in throughout. These are the cases that were wrong
 * before the store existed.
 */
describe("the initial load racing live updates", () => {
    it("does not let a stale batch take a tile back from a live update", () => {
        const store = new TileOwnership(10)

        store.applyUpdates([update(5, "jp")])              // someone claims tile 5
        const changes = store.applyBatch(batch({5: "fr"})) // the batch was already in flight

        expect(changes).toEqual([])
        expect(store.ownerOf(5)).toBe("jp")
        expect(counts(store)).toEqual({jp: 1})
    })

    it("keeps the live claim through every later batch", () => {
        const store = new TileOwnership(10)
        store.applyUpdates([update(5, "jp")])

        store.applyBatch(batch({5: "fr"}))
        store.applyBatch(batch({5: "de"}))

        expect(store.ownerOf(5)).toBe("jp")
        expect(counts(store)).toEqual({jp: 1})
    })

    it("counts a claimed tile once, not once per source", () => {
        const store = new TileOwnership(10)
        store.applyUpdates([update(1, "fr")])
        store.applyBatch(batch({1: "fr", 2: "fr"}))
        expect(counts(store)).toEqual({fr: 2})
    })

    it("still applies the batch to tiles nobody has claimed", () => {
        const store = new TileOwnership(10)
        store.applyUpdates([update(5, "jp")])
        expect(store.applyBatch(batch({5: "fr", 6: "fr"}))).toEqual([{tile: 6, country: "fr"}])
    })
})

describe("tile ids outside the map", () => {
    it("ignores ids the geometry has no point for", () => {
        const store = new TileOwnership(10)
        store.applyUpdates([update(0, "fr"), update(11, "fr"), update(-1, "fr")])
        store.applyBatch(batch({0: "jp", 99: "jp"}))
        expect(counts(store)).toEqual({})
    })

    it("accepts both ends of the range", () => {
        const store = new TileOwnership(10)
        store.applyUpdates([update(1, "fr"), update(10, "fr")])
        expect(counts(store)).toEqual({fr: 2})
    })
})

describe("counts", () => {
    it("stays consistent with the map across a long mixed run", () => {
        const store = new TileOwnership(50)
        const codes = ["fr", "jp", "de", "br"]

        for (let i = 0; i < 400; i++) {
            const tile = (i * 7) % 50 + 1
            const country = codes[i % codes.length]
            if (i % 3 === 0) {
                store.applyBatch(batch({[tile]: country}))
            } else {
                store.applyUpdates([update(tile, country)])
            }
        }

        const expected = new Map<string, number>()
        for (let tile = 1; tile <= 50; tile++) {
            const owner = store.ownerOf(tile)
            if (owner) expected.set(owner, (expected.get(owner) ?? 0) + 1)
        }

        expect(counts(store)).toEqual(Object.fromEntries(expected))
        const total = [...store.counts().values()].reduce((a, b) => a + b, 0)
        expect(total).toBeLessThanOrEqual(50)
    })
})
