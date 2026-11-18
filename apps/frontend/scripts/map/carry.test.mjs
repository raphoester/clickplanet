import fs from "node:fs"
import path from "node:path"

import {beforeAll, describe, expect, it} from "vitest"

import {EARTH_1, EARTH_2, boardDir, readBoard} from "./board.mjs"
import {carryStats} from "./carryStats.mjs"
import {COAST_REACH, carriesDir, carryNamed, decodeCarry, encodeCarry, parentsOf} from "./carry.mjs"

describe("the carry-over's format", () => {
    it("reads back what it wrote: no parent, a first one, jumps either way", () => {
        const parents = Uint32Array.from([0, 7, 7, 8, 0, 260503, 1, 0, 2 ** 31])
        const bytes = encodeCarry({positionsHash: 1, tiles: 2 ** 31}, {positionsHash: 4294967295, tiles: parents.length}, parents)
        expect(decodeCarry(bytes)).toEqual({fromHash: 1, toHash: 4294967295, fromTiles: 2 ** 31, toTiles: parents.length, parents})
    })

    it("refuses a parent past the old board, and a file cut short", () => {
        expect(() => encodeCarry({positionsHash: 1, tiles: 3}, {positionsHash: 2, tiles: 1}, Uint32Array.from([4]))).toThrow()
        const bytes = encodeCarry({positionsHash: 1, tiles: 300}, {positionsHash: 2, tiles: 2}, Uint32Array.from([200, 100]))
        expect(() => decodeCarry(bytes.subarray(0, bytes.length - 1))).toThrow()
    })
})

describe("earth-1 to earth-2", () => {
    let from, to, carry

    beforeAll(() => {
        from = readBoard(boardDir(EARTH_1.name))
        to = readBoard(boardDir(EARTH_2.name))
        carry = decodeCarry(fs.readFileSync(path.join(carriesDir, carryNamed(EARTH_1.name, EARTH_2.name))))
    }, 60_000)

    it("names both boards by their positions' hash and tile count", () => {
        expect(carry.fromHash).toBe(from.manifest.positionsHash)
        expect(carry.toHash).toBe(to.manifest.positionsHash)
        expect(carry.fromTiles).toBe(from.count)
        expect(carry.toTiles).toBe(to.count)
    })

    it("is what the generator writes from the boards today", () => {
        expect(parentsOf(from, to)).toEqual(carry.parents)
    }, 120_000)

    it("puts every parent within reach of its child: 0.62 of the old spacing at most", () => {
        let most = 0
        carry.parents.forEach((parent, t) => {
            if (parent === 0) return
            const chord = Math.hypot(
                to.positions[t * 3] - from.positions[(parent - 1) * 3],
                to.positions[t * 3 + 1] - from.positions[(parent - 1) * 3 + 1],
                to.positions[t * 3 + 2] - from.positions[(parent - 1) * 3 + 2],
            )
            most = Math.max(most, 2 * Math.asin(chord / 2) / from.manifest.spacing)
        })
        expect(most).toBeGreaterThan(COAST_REACH)
        expect(most).toBeLessThan(0.63)
    })

    it("gives a parent about 3.9 children inland, and carries all but about 1% of the new tiles", () => {
        const stats = carryStats(from, to, carry.parents)
        expect(stats.carried / to.count).toBeGreaterThan(0.99)
        expect(stats.inlandChildren).toBeGreaterThan(3.8)
        expect(stats.inlandChildren).toBeLessThan(3.9)
        expect(stats.orphans).toBeLessThan(100)
        expect(stats.blankNear).toBe(0)
    }, 60_000)
})
