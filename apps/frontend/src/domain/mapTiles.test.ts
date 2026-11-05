import {describe, expect, it} from "vitest"
import type {MapVersion, Ownerships, Update} from "../backends/backend.ts"
import {MapClock} from "./mapClock.ts"
import {MapTiles, TileChanges, tileRanges} from "./mapTiles.ts"

const START = 1_790_000_000

const at = (seq: number, epoch = START): MapVersion => ({epoch, seq})

// Two ranges: tiles 1 to 4, and 5 to 6.
const tilesOf = () => new MapTiles(6, 4, new MapClock())

const changes = (): TileChanges => ({owners: [], shields: []})

const batch = (start: number, end: number, version: MapVersion, owners: Record<number, string>, shields: Record<number, number> = {}): Ownerships => ({
    range: {start, end},
    version,
    bindings: new Map(Object.entries(owners).map(([tile, owner]) => [Number(tile), owner])),
    shields: new Map(Object.entries(shields).map(([tile, count]) => [Number(tile), count])),
})

const update = (tile: number, newCountry: string | undefined, version: MapVersion, shields = 0): Update =>
    ({tile, newCountry, previousCountry: undefined, clicked: true, shields, version})

const read = (tiles: MapTiles, ownerships: Ownerships) => {
    const taken = changes()
    const current = tiles.read(ownerships, taken)
    return {current, ...taken}
}

const live = (tiles: MapTiles, ...updates: Update[]) => {
    const taken = changes()
    tiles.applyUpdates(updates, taken)
    return taken
}

describe("tileRanges", () => {
    it("cuts the map into inclusive ranges that start at 1 and stop at its last tile", () => {
        expect(tileRanges(6, 4)).toEqual([{start: 1, end: 4}, {start: 5, end: 6}])
        expect(tileRanges(8, 4)).toEqual([{start: 1, end: 4}, {start: 5, end: 8}])
    })
})

describe("a tile takes a value only when its version is newer", () => {
    it("keeps what the stream said over a read that was older", () => {
        const tiles = tilesOf()
        tiles.opened(at(10))
        live(tiles, update(1, "jp", at(12)))

        const {owners} = read(tiles, batch(1, 4, at(11), {1: "fr", 2: "fr"}))

        expect(owners).toEqual([{tile: 2, country: "fr"}])
        expect(tiles.ownership.ownerOf(1)).toBe("jp")
    })

    it("takes a read newer than what the stream said, and empties what it does not list", () => {
        const tiles = tilesOf()
        tiles.opened(at(10))
        live(tiles, update(1, "jp", at(11)), update(2, "jp", at(11)))

        read(tiles, batch(1, 4, at(13), {1: "fr"}))

        expect(tiles.ownership.ownerOf(1)).toBe("fr")
        expect(tiles.ownership.ownerOf(2)).toBeUndefined()
    })

    it("drops a stream update older than the read already taken", () => {
        const tiles = tilesOf()
        tiles.opened(at(10))
        read(tiles, batch(1, 4, at(14), {1: "fr"}))

        expect(live(tiles, update(1, "jp", at(13))).owners).toEqual([])
        expect(tiles.ownership.ownerOf(1)).toBe("fr")
    })

    it("lets a later update win over a bomb's clear that waited for its impact", () => {
        const tiles = tilesOf()
        tiles.opened(at(10))
        read(tiles, batch(1, 4, at(10), {1: "fr", 2: "fr"}))
        live(tiles, update(1, "jp", at(12)))

        const landed = changes()
        tiles.applyBlast([1, 2], [], at(11), landed)

        expect(landed.owners).toEqual([{tile: 2, country: undefined}])
        expect(tiles.ownership.ownerOf(1)).toBe("jp")
    })

    it("sets the shields a struck tile kept, however many times the strike is heard", () => {
        const tiles = tilesOf()
        tiles.opened(at(10))
        read(tiles, batch(1, 4, at(10), {3: "fr"}, {3: 3}))

        tiles.applyBlast([], [{tile: 3, shields: 2}], at(11), changes())
        tiles.applyBlast([], [{tile: 3, shields: 2}], at(11), changes())

        expect(tiles.shields.shieldsOf(3)).toBe(2)
    })

    it("knows a tile once anything brought it a value", () => {
        const tiles = tilesOf()
        tiles.opened(at(10))
        live(tiles, update(5, "jp", at(11)))

        expect(tiles.known(5)).toBe(true)
        expect(tiles.known(6)).toBe(false)
    })
})

describe("what a read misses", () => {
    it("asks again for a range read before the stream's opening, or never read", () => {
        const tiles = tilesOf()
        read(tiles, batch(1, 4, at(9), {}))
        tiles.opened(at(10))

        expect(tiles.stale(tiles.ranges)).toEqual([{start: 1, end: 4}, {start: 5, end: 6}])

        expect(read(tiles, batch(1, 4, at(10), {})).current).toBe(true)
        expect(read(tiles, batch(5, 6, at(11), {})).current).toBe(true)
        expect(tiles.stale(tiles.ranges)).toEqual([])
        expect(tiles.settled()).toBe(true)
    })

    it("asks for every range again when the stream opens again past them, and for none when nothing moved", () => {
        const tiles = tilesOf()
        tiles.opened(at(10))
        read(tiles, batch(1, 4, at(10), {}))
        read(tiles, batch(5, 6, at(12), {}))

        tiles.opened(at(12))

        expect(tiles.stale(tiles.ranges)).toEqual([{start: 1, end: 4}])
    })

    it("reads nothing when the stream opens again with no write since the last thing it told", () => {
        const tiles = tilesOf()
        tiles.opened(at(10))
        read(tiles, batch(1, 4, at(10), {}))
        read(tiles, batch(5, 6, at(11), {}))
        live(tiles, update(2, "jp", at(14)))

        tiles.opened(at(14))
        expect(tiles.stale(tiles.ranges)).toEqual([])

        tiles.opened(at(15))
        expect(tiles.stale(tiles.ranges)).toHaveLength(2)
    })

    it("counts the share of the ranges read at least once", () => {
        const tiles = tilesOf()

        expect(tiles.readShare()).toBe(0)
        read(tiles, batch(5, 6, at(3), {}))
        expect(tiles.readShare()).toBe(0.5)
    })
})

describe("a server that started again", () => {
    it("forgets every version: a read of the new start replaces what the old one said, and the tiles stay drawn until then", () => {
        const tiles = tilesOf()
        tiles.opened(at(500))
        read(tiles, batch(1, 4, at(500), {1: "fr"}))

        expect(tiles.opened(at(3, START + 60))).toEqual({restarted: true})
        expect(tiles.ownership.ownerOf(1)).toBe("fr")
        expect(tiles.known(1)).toBe(false)
        expect(tiles.stale(tiles.ranges)).toHaveLength(2)

        read(tiles, batch(1, 4, at(3, START + 60), {1: "jp"}))
        expect(tiles.ownership.ownerOf(1)).toBe("jp")
    })

    it("takes no read nor update of another start", () => {
        const tiles = tilesOf()
        tiles.opened(at(10, START + 60))

        expect(read(tiles, batch(1, 4, at(900), {1: "fr"}))).toEqual({current: false, owners: [], shields: []})
        expect(live(tiles, update(2, "fr", at(901))).owners).toEqual([])
    })
})

describe("a fortify adds a shield to each tile of the landmass its flag holds", () => {
    const landmass = [1, 2, 5]

    it("raises at once the tiles read at the stream's opening or after, up to the most", () => {
        const tiles = tilesOf()
        tiles.opened(at(10))
        read(tiles, batch(1, 4, at(10), {1: "fr", 2: "fr"}, {2: 3}))
        read(tiles, batch(5, 6, at(10), {5: "fr"}))

        const raised = tiles.applyFortification(landmass, "fr", 3, at(11))

        expect(raised).toEqual([{tile: 1, shields: 1, was: 0}, {tile: 5, shields: 1, was: 0}])
        expect(tiles.shields.shieldsOf(2)).toBe(3)
    })

    it("raises a tile once, and not a tile whose read already holds it", () => {
        const tiles = tilesOf()
        tiles.opened(at(10))
        read(tiles, batch(1, 4, at(10), {1: "fr", 2: "fr"}))
        read(tiles, batch(5, 6, at(12), {5: "fr"}, {5: 1}))

        tiles.applyFortification(landmass, "fr", 3, at(11))
        tiles.applyFortification(landmass, "fr", 3, at(11))

        expect([1, 2, 5].map((tile) => tiles.shields.shieldsOf(tile))).toEqual([1, 1, 1])
    })

    it("waits for the read of a range older than the stream, then raises what that read does not hold", () => {
        const tiles = tilesOf()
        tiles.opened(at(10))
        read(tiles, batch(1, 4, at(10), {1: "fr", 2: "fr"}))
        read(tiles, batch(5, 6, at(8), {5: "de"}))

        expect(tiles.applyFortification(landmass, "fr", 3, at(12))).toHaveLength(2)
        expect(tiles.shields.shieldsOf(5)).toBe(0)

        const reread = read(tiles, batch(5, 6, at(11), {5: "fr"}))

        expect(reread.current).toBe(true)
        expect(reread.shields).toEqual([{tile: 5, shields: 1, was: 0}])
    })

    it("does not raise a waiting tile twice when its read came after the fortify", () => {
        const tiles = tilesOf()
        tiles.opened(at(10))
        read(tiles, batch(1, 4, at(10), {1: "fr", 2: "fr"}))
        tiles.applyFortification(landmass, "fr", 3, at(12))

        read(tiles, batch(5, 6, at(13), {5: "fr"}, {5: 1}))

        expect(tiles.shields.shieldsOf(5)).toBe(1)
    })
})
