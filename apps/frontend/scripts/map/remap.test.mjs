import {describe, expect, it} from "vitest"

import {remap, remapLandmasses, remapSQL} from "./remap.mjs"

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
        const sql = remapSQL(plan)
        expect(sql).toContain("DELETE FROM tiles;")
        expect(sql).toContain("INSERT INTO tiles SELECT * FROM tiles_remapped;")
        expect(sql).not.toMatch(/UPDATE tiles\b/)
    })

    it("carries every column of a tile, its shields and holder and whatever comes after", () => {
        const sql = remapSQL(plan)
        expect(sql).toContain("SELECT t.*, r.to_id + (t.id - r.from_id) AS remapped_id\n")
        expect(sql).toContain("ALTER TABLE tiles_remapped DROP COLUMN remapped_id;")
        expect(sql).not.toMatch(/t\.shields|t\.holder/)
    })

    it("deletes the ledger's takes on tiles the new map does not have", () => {
        expect(remapSQL(plan)).toContain("DELETE FROM ledger_events e\nWHERE e.payload IS NULL")
    })

    it("keeps every event with a payload, and moves every tile its payload lists", () => {
        const sql = remapSQL(plan)
        expect(sql).toContain("SET tile = 0, previous = ''\nWHERE e.payload IS NOT NULL")
        for (const key of ["cleared", "taken", "struck"]) {
            expect(sql).toContain(`jsonb_set(e.payload, '{${key}}'`)
            expect(sql).toContain(`WHERE e.payload ? '${key}';`)
        }
        expect(sql).not.toMatch(/e\.kind/)
    })

    it("walks the runs the other way going down", () => {
        expect(remapSQL(plan, {down: true})).toContain("from coordinates-new.bin to coordinates-old.bin")
    })

    it("leaves the landmasses alone without a borders blob to follow them to", () => {
        expect(remapSQL(plan)).not.toContain("landmass")
    })

    const followed = {
        ...plan,
        landmasses: [{from: 1, to: 1}, {from: 2, to: 3}],
        borders: {from: "borders-old.bin", to: "borders-new.bin"},
    }

    it("moves the landmass a fortify names, and forgets one the new blob has no landmass for", () => {
        const sql = remapSQL(followed)
        expect(sql).toContain("    (1, 1),\n    (2, 3);")
        expect(sql).toContain("jsonb_set(e.payload, '{landmass}', to_jsonb(COALESCE((")
        expect(sql).toContain("), 0)))\nWHERE e.payload ? 'landmass';")
        expect(sql).not.toMatch(/e\.kind/)
    })

    it("copies the fortify locks to the new blob's name, and back going down", () => {
        expect(remapSQL(followed)).toContain("SELECT l.map, 'borders-new.bin', r.to_id, l.fortified_by")
        expect(remapSQL(followed)).toContain("WHERE l.asset = 'borders-old.bin'")
        expect(remapSQL(followed)).toContain("ON CONFLICT (map, asset, id)")
        const down = remapSQL(followed, {down: true})
        expect(down).toContain("    (1, 1),\n    (3, 2);")
        expect(down).toContain("SELECT l.map, 'borders-old.bin', r.to_id, l.fortified_by")
        expect(down).toContain("WHERE l.asset = 'borders-new.bin'")
    })
})

describe("remapLandmasses", () => {
    const blob = (codes, landmass) => ({codes, landmass: Uint16Array.from(landmass)})

    it("follows each landmass to the one its tiles are in", () => {
        const before = blob(["", "fr", "it"], [1, 1, 2, 2])
        const after = blob(["", "fr", "nr", "it"], [1, 1, 2, 3, 3])
        const {runs} = remap(["a", "b", "c", "d"], ["a", "b", "new", "c", "d"])

        expect(remapLandmasses(before, after, runs)).toEqual([{from: 1, to: 1}, {from: 2, to: 2 + 1}])
    })

    it("follows a landmass that gave a tile to another country", () => {
        const before = blob(["", "it"], [1, 1, 1])
        const after = blob(["", "it", "va"], [1, 2, 1])
        const {runs} = remap(["a", "b", "c"], ["a", "b", "c"])

        expect(remapLandmasses(before, after, runs)).toEqual([{from: 1, to: 1}])
    })

    it("follows a landmass that split to the piece that holds most of it", () => {
        const {runs} = remap(["a", "b", "c", "d"], ["a", "b", "c", "d"])
        expect(remapLandmasses(blob(["", "ca"], [1, 1, 1, 1]), blob(["", "ca", "ca"], [1, 1, 1, 2]), runs))
            .toEqual([{from: 1, to: 1}])
    })

    it("follows neither half of a landmass split in two equal pieces, nor two that merged", () => {
        const {runs} = remap(["a", "b", "c", "d"], ["a", "b", "c", "d"])
        expect(remapLandmasses(blob(["", "fr"], [1, 1, 1, 1]), blob(["", "fr", "fr"], [1, 1, 2, 2]), runs)).toEqual([])
        expect(remapLandmasses(blob(["", "fr", "fr"], [1, 1, 2, 2]), blob(["", "fr"], [1, 1, 1, 1]), runs)).toEqual([])
    })

    it("follows nothing for a landmass whose every tile went", () => {
        const {runs} = remap(["a", "b", "lake"], ["a", "b"])
        expect(remapLandmasses(blob(["", "tz", "mw"], [1, 1, 2]), blob(["", "tz"], [1, 1]), runs))
            .toEqual([{from: 1, to: 1}])
    })
})
