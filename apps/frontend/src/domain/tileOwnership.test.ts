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

    it("ignores a previous owner that disagrees with what it holds", () => {
        const store = new TileOwnership(10)
        store.applyUpdates([update(1, "fr")])
        store.applyUpdates([update(1, "jp", "de")])
        expect(counts(store)).toEqual({jp: 1})
    })

    it("never lets a count go negative", () => {
        const store = new TileOwnership(10)
        store.applyUpdates([update(1, "jp", "fr")])
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

describe("the initial load racing live updates", () => {
    it("does not let a stale batch take a tile back from a live update", () => {
        const store = new TileOwnership(10)

        store.applyUpdates([update(5, "jp")])
        const changes = store.applyBatch(batch({5: "fr"}))

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

// A click paints before the server has agreed to it, and the server can still
// refuse — a spent bucket, a blocked VPN, a session it would not mint. What was
// painted has to go back, or the player keeps looking at tiles nobody gave them.
describe("optimistic clicks", () => {
    it("paints the tile and counts it while the click is in flight", () => {
        const store = new TileOwnership(10)

        const {changes, claim} = store.applyOptimistic(1, "fr")

        expect(changes).toEqual([{tile: 1, country: "fr"}])
        expect(claim).toEqual({tile: 1, token: expect.any(Number)})
        expect(store.ownerOf(1)).toBe("fr")
        expect(counts(store)).toEqual({fr: 1})
    })

    it("takes the tile back to unowned when the click is refused", () => {
        const store = new TileOwnership(10)
        const {claim} = store.applyOptimistic(1, "fr")

        expect(store.rollback(claim)).toEqual([{tile: 1, country: undefined}])
        expect(store.ownerOf(1)).toBeUndefined()
        expect(counts(store)).toEqual({})
    })

    it("gives the tile back to whoever held it before", () => {
        const store = new TileOwnership(10)
        store.applyBatch(batch({1: "jp"}))

        const {claim} = store.applyOptimistic(1, "fr")
        expect(counts(store)).toEqual({fr: 1})

        expect(store.rollback(claim)).toEqual([{tile: 1, country: "jp"}])
        expect(store.ownerOf(1)).toBe("jp")
        expect(counts(store)).toEqual({jp: 1})
    })

    it("reports nothing to repaint when the click only repeated the owner", () => {
        const store = new TileOwnership(10)
        store.applyUpdates([update(1, "fr")])

        const {changes, claim} = store.applyOptimistic(1, "fr")
        expect(changes).toEqual([])

        expect(store.rollback(claim)).toEqual([])
        expect(store.ownerOf(1)).toBe("fr")
        expect(counts(store)).toEqual({fr: 1})
    })

    it("ignores a click on a tile the geometry has no point for", () => {
        const store = new TileOwnership(10)

        const {changes, claim} = store.applyOptimistic(11, "fr")

        expect(changes).toEqual([])
        expect(claim).toBeUndefined()
        expect(store.rollback(claim)).toEqual([])
        expect(counts(store)).toEqual({})
    })
})

describe("a refusal arriving after the tile has moved on", () => {
    it("leaves the tile alone once the server has echoed the click", () => {
        const store = new TileOwnership(10)
        const {claim} = store.applyOptimistic(1, "fr")

        store.applyUpdates([update(1, "fr")])

        expect(store.rollback(claim)).toEqual([])
        expect(store.ownerOf(1)).toBe("fr")
        expect(counts(store)).toEqual({fr: 1})
    })

    it("leaves the tile alone once someone else has claimed it", () => {
        const store = new TileOwnership(10)
        const {claim} = store.applyOptimistic(1, "fr")

        store.applyUpdates([update(1, "jp", "fr")])

        expect(store.rollback(claim)).toEqual([])
        expect(store.ownerOf(1)).toBe("jp")
        expect(counts(store)).toEqual({jp: 1})
    })

    it("rolls back once, however many times the refusal is reported", () => {
        const store = new TileOwnership(10)
        store.applyBatch(batch({1: "jp"}))
        const {claim} = store.applyOptimistic(1, "fr")

        store.rollback(claim)
        expect(store.rollback(claim)).toEqual([])
        expect(counts(store)).toEqual({jp: 1})
    })

    it("keeps the paint of a later click still in flight", () => {
        const store = new TileOwnership(10)
        const first = store.applyOptimistic(1, "fr").claim
        store.applyOptimistic(1, "jp")

        expect(store.rollback(first)).toEqual([])
        expect(store.ownerOf(1)).toBe("jp")
        expect(counts(store)).toEqual({jp: 1})
    })

    it("falls back to an earlier click still in flight", () => {
        const store = new TileOwnership(10)
        store.applyOptimistic(1, "fr")
        const second = store.applyOptimistic(1, "jp").claim

        expect(store.rollback(second)).toEqual([{tile: 1, country: "fr"}])
        expect(store.ownerOf(1)).toBe("fr")
        expect(counts(store)).toEqual({fr: 1})
    })

    it("unwinds a whole refused burst back to where the tile started", () => {
        const store = new TileOwnership(10)
        store.applyBatch(batch({1: "de"}))

        const claims = ["fr", "jp", "br"].map((code) => store.applyOptimistic(1, code).claim)
        for (const claim of claims) store.rollback(claim)

        expect(store.ownerOf(1)).toBe("de")
        expect(counts(store)).toEqual({de: 1})
    })
})

describe("a rolled-back click and the initial load", () => {
    it("falls back to a batch that landed while the click was in flight", () => {
        const store = new TileOwnership(10)
        const {claim} = store.applyOptimistic(5, "fr")

        expect(store.applyBatch(batch({5: "jp"}))).toEqual([])
        expect(store.ownerOf(5)).toBe("fr")

        expect(store.rollback(claim)).toEqual([{tile: 5, country: "jp"}])
        expect(counts(store)).toEqual({jp: 1})
    })

    it("still refuses a batch for a tile a live update had already claimed", () => {
        const store = new TileOwnership(10)
        store.applyUpdates([update(5, "jp")])
        const {claim} = store.applyOptimistic(5, "fr")

        store.applyBatch(batch({5: "de"}))

        expect(store.rollback(claim)).toEqual([{tile: 5, country: "jp"}])
        expect(store.ownerOf(5)).toBe("jp")
        expect(counts(store)).toEqual({jp: 1})
    })

    it("lets a later batch paint a tile whose only claim was rolled back", () => {
        const store = new TileOwnership(10)
        const {claim} = store.applyOptimistic(5, "fr")
        store.rollback(claim)

        expect(store.applyBatch(batch({5: "jp"}))).toEqual([{tile: 5, country: "jp"}])
        expect(store.ownerOf(5)).toBe("jp")
    })

    it("keeps holding off batches while a rolled-back tile has a click left in flight", () => {
        const store = new TileOwnership(10)
        const first = store.applyOptimistic(5, "fr").claim
        store.applyOptimistic(5, "jp")

        store.rollback(first)

        expect(store.applyBatch(batch({5: "de"}))).toEqual([])
        expect(store.ownerOf(5)).toBe("jp")
    })
})

describe("counts after rollbacks", () => {
    it("stays consistent with the map across a long mixed run", () => {
        const store = new TileOwnership(50)
        const codes = ["fr", "jp", "de", "br"]
        const inFlight: (ReturnType<typeof store.applyOptimistic>["claim"])[] = []

        for (let i = 0; i < 400; i++) {
            const tile = (i * 7) % 50 + 1
            const country = codes[i % codes.length]

            if (i % 4 === 0) store.applyBatch(batch({[tile]: country}))
            else if (i % 4 === 1) store.applyUpdates([update(tile, country)])
            else inFlight.push(store.applyOptimistic(tile, country).claim)

            if (i % 3 === 0) store.rollback(inFlight.shift())
        }
        for (const claim of inFlight) store.rollback(claim)

        const expected = new Map<string, number>()
        for (let tile = 1; tile <= 50; tile++) {
            const owner = store.ownerOf(tile)
            if (owner) expected.set(owner, (expected.get(owner) ?? 0) + 1)
        }

        expect(counts(store)).toEqual(Object.fromEntries(expected))
        expect([...store.counts().values()].every((count) => count > 0)).toBe(true)
    })
})
