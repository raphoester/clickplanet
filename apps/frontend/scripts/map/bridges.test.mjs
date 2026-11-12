import {describe, expect, it} from "vitest"

import {bridgesOf, piecesApart} from "./bridges.mjs"
import {positionOf} from "./seats.mjs"

// Tiles on the equator at the longitudes given, then one sea vertex that every tile in `coast` touches; `land` pairs
// tiles that touch.
const mapOf = (longitudes, {land = [], coast = longitudes.map((_, t) => t)} = {}) => {
    const sea = longitudes.length
    const around = Array.from({length: sea + 1}, () => [])
    for (const [a, b] of [...land, ...coast.map((t) => [t, sea])]) {
        around[a].push(b)
        around[b].push(a)
    }
    const at = new Uint32Array(around.length + 1)
    around.forEach((next, v) => { at[v + 1] = at[v] + next.length })
    return {
        positions: Float64Array.from(longitudes.flatMap((lon) => positionOf([lon, 0]))),
        at,
        to: Uint32Array.from(around.flat()),
        tileOf: Int32Array.from([...longitudes.map((_, t) => t), -1]),
    }
}

describe("bridgesOf", () => {
    it("joins two pieces when no third is nearer to both", () => {
        const {bridges, pieces, stones} = bridgesOf({...mapOf([0, 10, 20]), capitals: {}}, {})
        expect(pieces).toBe(3)
        expect(stones).toBe(2)
        expect(bridges).toEqual([[1, 2], [2, 3]])
    })

    it("joins two pieces far apart when nothing lies between", () => {
        const {bridges} = bridgesOf({...mapOf([0, 10, 100]), capitals: {}}, {})
        expect(bridges).toEqual([[1, 2], [2, 3]])
    })

    it("lands on a capital only when its piece has no other coast", () => {
        const map = mapOf([5, 0, 10], {land: [[0, 1]]})
        expect(bridgesOf({...map, capitals: {aa: 1, bb: 3}}, {}).bridges).toEqual([[2, 3]])
    })

    it("adds each sea lane, snapped to the nearest coast that is no capital", () => {
        const map = mapOf([0, 1, 10, 20])
        const lanes = {"a lane": [[0.4, 0.5], [19.5, -0.5]]}
        expect(bridgesOf({...map, capitals: {aa: 1}}, lanes).bridges).toContainEqual([2, 4])
    })

    it("refuses a lane whose point is far from every coast", () => {
        const lanes = {"lost at sea": [[0, 0], [60, 0]]}
        expect(() => bridgesOf({...mapOf([0, 10]), capitals: {}}, lanes)).toThrow(/lost at sea: \[60,0] is \d+ km/)
    })
})

describe("piecesApart", () => {
    it("counts the pieces no bridge reaches", () => {
        const {at, to, tileOf} = mapOf([0, 10, 20], {land: [[0, 1]]})
        expect(piecesApart({at, to, tileOf}, 3, [])).toBe(1)
        expect(piecesApart({at, to, tileOf}, 3, [[2, 3]])).toBe(0)
    })
})
