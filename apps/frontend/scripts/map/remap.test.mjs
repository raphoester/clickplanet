import {describe, expect, it} from "vitest"

import {remap, remapSQL} from "./remap.mjs"

// Positions stand in for the keys `keyOf` writes; what matters is only which are in both lists.
const ids = (...names) => names

describe("remap", () => {
    it("is the identity when nothing moved", () => {
        const {runs, kept, removed, added} = remap(ids("a", "b", "c"), ids("a", "b", "c"))
        expect(runs).toEqual([{from: 1, to: 1, span: 3}])
        expect({kept, removed, added}).toEqual({kept: 3, removed: [], added: 0})
    })

    it("shifts everything after a tile that went", () => {
        const {runs, kept, removed, added} = remap(ids("a", "b", "c", "d"), ids("a", "c", "d"))
        expect(runs).toEqual([{from: 1, to: 1, span: 1}, {from: 3, to: 2, span: 2}])
        expect({kept, removed, added}).toEqual({kept: 3, removed: [2], added: 0})
    })

    it("shifts everything after a tile that arrived", () => {
        const {runs, kept, removed, added} = remap(ids("a", "b"), ids("a", "new", "b"))
        expect(runs).toEqual([{from: 1, to: 1, span: 1}, {from: 2, to: 3, span: 1}])
        expect({kept, removed, added}).toEqual({kept: 2, removed: [], added: 1})
    })

    it("joins a stretch that moves by one offset into one run", () => {
        const before = ids("a", "b", "c", "d", "e")
        const after = ids("x", "a", "b", "c", "d", "e")
        expect(remap(before, after).runs).toEqual([{from: 1, to: 2, span: 5}])
    })

    // The whole reason the mapping fits in a migration: both blobs are the same lattice in the same
    // generation order, so a tile in both cannot come out before one it used to come after. A blob
    // built some other way would silently produce a mapping that overwrites rows, so it is refused.
    it("refuses a mapping that is not monotonic", () => {
        expect(() => remap(ids("a", "b"), ids("b", "a"))).toThrow(/not monotonic/)
    })

    it("counts a tile that went and one that arrived apart", () => {
        const {kept, removed, added} = remap(ids("a", "gone", "b"), ids("a", "new", "b"))
        expect({kept, removed, added}).toEqual({kept: 2, removed: [2], added: 1})
    })
})

describe("remapSQL", () => {
    const plan = {
        ...remap(ids("a", "gone", "b"), ids("a", "new", "b")),
        before: 3,
        after: 3,
        from: "coordinates-old.bin",
        to: "coordinates-new.bin",
    }

    it("names both blobs and carries every run", () => {
        const sql = remapSQL(plan)
        expect(sql).toContain("from coordinates-old.bin to coordinates-new.bin")
        expect(sql).toContain("    (1, 1, 1),\n    (3, 3, 1);")
    })

    it("rebuilds tiles rather than updating in place", () => {
        // An id can move up or down, so an UPDATE would collide with a row it has not moved yet.
        const sql = remapSQL(plan)
        expect(sql).toContain("DELETE FROM tiles;")
        expect(sql).toContain("INSERT INTO tiles (id, country)")
        expect(sql).not.toMatch(/UPDATE tiles\b/)
    })

    it("deletes the ledger's takes on tiles the new map does not have", () => {
        expect(remapSQL(plan)).toContain("DELETE FROM ledger_takes")
    })

    it("walks the runs the other way going down", () => {
        expect(remapSQL(plan, {down: true})).toContain("from coordinates-new.bin to coordinates-old.bin")
    })
})
