import {describe, expect, it} from "vitest"
import {OwnerChange, TileOwnership} from "./tileOwnership.ts"

const read = (store: TileOwnership, tile: number, country: string | undefined) => {
    const changes: OwnerChange[] = []
    store.take(tile, country, changes)
    return changes
}

const live = (store: TileOwnership, tile: number, country: string | undefined) => {
    const changes: OwnerChange[] = []
    store.set(tile, country, changes)
    return changes
}

describe("a tile's flag", () => {
    it("is what a read or the stream sets, and only a change is reported", () => {
        const store = new TileOwnership(10)

        expect(read(store, 1, "fr")).toEqual([{tile: 1, country: "fr"}])
        expect(read(store, 1, "fr")).toEqual([])
        expect(live(store, 1, "jp")).toEqual([{tile: 1, country: "jp"}])
        expect(live(store, 1, undefined)).toEqual([{tile: 1, country: undefined}])
        expect(store.ownerOf(1)).toBeUndefined()
    })

    it("ignores ids the geometry has no point for", () => {
        const store = new TileOwnership(10)

        expect([...read(store, 0, "fr"), ...live(store, 11, "fr"), ...live(store, -1, "fr")]).toEqual([])
        expect(read(store, 10, "fr")).toEqual([{tile: 10, country: "fr"}])
    })
})

describe("optimistic clicks", () => {
    it("paints the tile while the click is in flight", () => {
        const store = new TileOwnership(10)

        const {changes, claim} = store.applyOptimistic(1, "fr")

        expect(changes).toEqual([{tile: 1, country: "fr"}])
        expect(claim).toEqual({tile: 1, token: expect.any(Number)})
        expect(store.ownerOf(1)).toBe("fr")
    })

    it("gives the tile back to whoever held it before when the click is refused", () => {
        const store = new TileOwnership(10)
        read(store, 1, "jp")

        const {claim} = store.applyOptimistic(1, "fr")

        expect(store.rollback(claim)).toEqual([{tile: 1, country: "jp"}])
        expect(store.ownerOf(1)).toBe("jp")
    })

    it("reports nothing to repaint when the click only repeated the owner", () => {
        const store = new TileOwnership(10)
        live(store, 1, "fr")

        const {changes, claim} = store.applyOptimistic(1, "fr")

        expect(changes).toEqual([])
        expect(store.rollback(claim)).toEqual([])
        expect(store.ownerOf(1)).toBe("fr")
    })

    it("ignores a click on a tile the geometry has no point for", () => {
        const store = new TileOwnership(10)

        const {changes, claim} = store.applyOptimistic(11, "fr")

        expect(changes).toEqual([])
        expect(claim).toBeUndefined()
        expect(store.rollback(claim)).toEqual([])
    })

    it("keeps the paint of a later click still in flight, and falls back to an earlier one", () => {
        const store = new TileOwnership(10)
        const first = store.applyOptimistic(1, "fr").claim
        const second = store.applyOptimistic(1, "jp").claim

        expect(store.rollback(first)).toEqual([])
        expect(store.ownerOf(1)).toBe("jp")

        const third = store.applyOptimistic(1, "br").claim
        expect(store.rollback(third)).toEqual([{tile: 1, country: "jp"}])
        expect(store.rollback(second)).toEqual([{tile: 1, country: undefined}])
    })

    it("unwinds a whole refused burst back to where the tile started", () => {
        const store = new TileOwnership(10)
        read(store, 1, "de")

        const claims = ["fr", "jp", "br"].map((code) => store.applyOptimistic(1, code).claim)
        for (const claim of claims) store.rollback(claim)

        expect(store.ownerOf(1)).toBe("de")
    })
})

describe("a click in flight and what the server says", () => {
    it("stays painted under a read, and rolls back to what the read said", () => {
        const store = new TileOwnership(10)
        const {claim} = store.applyOptimistic(5, "fr")

        expect(read(store, 5, "jp")).toEqual([])
        expect(store.ownerOf(5)).toBe("fr")

        expect(store.rollback(claim)).toEqual([{tile: 5, country: "jp"}])
    })

    it("is settled by the stream: its echo or another flag's take, so a late refusal changes nothing", () => {
        const store = new TileOwnership(10)
        const echoed = store.applyOptimistic(1, "fr").claim
        const retaken = store.applyOptimistic(2, "fr").claim

        live(store, 1, "fr")
        expect(live(store, 2, "jp")).toEqual([{tile: 2, country: "jp"}])

        expect(store.rollback(echoed)).toEqual([])
        expect(store.rollback(retaken)).toEqual([])
        expect(store.ownerOf(1)).toBe("fr")
        expect(store.ownerOf(2)).toBe("jp")
    })

    it("rolls back once, however many times the refusal is reported", () => {
        const store = new TileOwnership(10)
        read(store, 1, "jp")
        const {claim} = store.applyOptimistic(1, "fr")

        store.rollback(claim)

        expect(store.rollback(claim)).toEqual([])
        expect(store.ownerOf(1)).toBe("jp")
    })
})
