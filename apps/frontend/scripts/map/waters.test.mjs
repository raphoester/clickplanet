import {describe, expect, it} from "vitest"

import {watersOf} from "./waters.mjs"

// A lattice of `land.length` vertices that touch where `pairs` say, as `neighbours` lays one out, and the tile
// index of each vertex, in lattice order, or -1 where it is water.
const lattice = (land, pairs) => {
    const around = land.map(() => [])
    for (const [a, b] of pairs) {
        around[a].push(b)
        around[b].push(a)
    }
    const at = new Uint32Array(land.length + 1)
    around.forEach((next, v) => { at[v + 1] = at[v] + next.length })
    let tile = 0
    return {
        edges: {at, to: Uint32Array.from(around.flat())},
        tileOf: Int32Array.from(land, (isLand) => isLand ? tile++ : -1),
    }
}

describe("watersOf", () => {
    it("keeps two waters that land cuts apart as two bodies, the longer coast first", () => {
        const {edges, tileOf} = lattice(
            [false, true, true, false, true, true],
            [[0, 1], [0, 2], [3, 2], [3, 4], [3, 5], [1, 2], [2, 4], [4, 5]],
        )
        expect(watersOf(edges, tileOf)).toEqual([[2, 3, 4], [1, 2]])
    })

    it("joins water that touches into one body", () => {
        const {edges, tileOf} = lattice([false, false, true, true], [[0, 1], [0, 2], [1, 3], [2, 3]])
        expect(watersOf(edges, tileOf)).toEqual([[1, 2]])
    })

    it("names tiles by their wire id, one past their index", () => {
        const {edges, tileOf} = lattice([true, true, false], [[0, 1], [1, 2]])
        expect(watersOf(edges, tileOf)).toEqual([[2]])
    })

    it("draws no water on a map of land", () => {
        const {edges, tileOf} = lattice([true, true, true], [[0, 1], [1, 2]])
        expect(watersOf(edges, tileOf)).toEqual([])
    })
})
