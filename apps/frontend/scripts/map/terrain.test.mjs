import {describe, expect, it} from "vitest"

import {LAND, OPEN_SEA, seaKind} from "../../src/domain/terrain.ts"
import {PHOTO_COLOURS, kindsFromNeighbours, landKinds, smoothed, terrainOf} from "./terrain.mjs"

const kindOf = (name) => LAND.indexOf(name)

// Points joined as the pairs say; `tiles` lists the points that are tiles, in tile order.
const graphOf = (count, pairs, tiles) => {
    const around = Array.from({length: count}, () => [])
    for (const [a, b] of pairs) {
        around[a].push(b)
        around[b].push(a)
    }
    const at = new Uint32Array(count + 1)
    around.forEach((next, v) => { at[v + 1] = at[v] + next.length })
    const tileOf = new Int32Array(count).fill(-1)
    tiles.forEach((point, t) => { tileOf[point] = t })
    return {lattice: {at, to: Uint32Array.from(around.flat()), tileOf}, points: Uint32Array.from(tiles)}
}

// A line of points, each joined to the next.
const line = (count, tiles) => graphOf(count, Array.from({length: count - 1}, (_, v) => [v, v + 1]), tiles)

describe("landKinds", () => {
    it("gives each kind's own photo colour that kind", () => {
        expect([...landKinds(LAND.map((kind) => PHOTO_COLOURS[kind]))]).toEqual(LAND.map((_, k) => k))
    })

    it("takes a yellow white for sand in the sun, not snow", () => {
        expect(landKinds([[96, 0, 4]])[0]).toBe(kindOf("snow"))
        expect(landKinds([[96, 0, 12]])[0]).not.toBe(kindOf("snow"))
    })
})

describe("smoothed", () => {
    it("gives a lone tile the kind all round it", () => {
        const {lattice, points} = graphOf(7, [1, 2, 3, 4, 5, 6].map((v) => [0, v]), [0, 1, 2, 3, 4, 5, 6])
        const kinds = Uint8Array.from([kindOf("sand"), 0, 0, 0, 0, 0, 0])
        expect(smoothed(kinds, lattice, points)[0]).toBe(kindOf("forest"))
    })

    it("keeps a tile's kind when only one neighbour differs, its own counting twice", () => {
        const {lattice, points} = line(3, [0, 1, 2])
        const kinds = Uint8Array.from([kindOf("grass"), kindOf("arid"), kindOf("arid")])
        expect([...smoothed(kinds, lattice, points)]).toEqual([...kinds])
    })

    it("leaves an island of one tile as it is", () => {
        const {lattice, points} = line(3, [1])
        expect([...smoothed(Uint8Array.from([kindOf("snow")]), lattice, points)]).toEqual([kindOf("snow")])
    })
})

describe("terrainOf", () => {
    it("bands the sea by its steps from land, out to the open sea", () => {
        const {lattice} = line(7, [0])
        const ids = Uint32Array.from([6, 5, 4, 3, 2, 1, 0])
        const terrain = terrainOf(Uint8Array.from([kindOf("tundra")]), lattice, ids)
        expect([...terrain].reverse()).toEqual([kindOf("tundra"), seaKind(1), seaKind(2), seaKind(3), OPEN_SEA, OPEN_SEA, OPEN_SEA])
    })

    it("measures a sea point from the nearest land", () => {
        const {lattice} = line(5, [0, 4])
        const terrain = terrainOf(Uint8Array.from([kindOf("grass"), kindOf("sand")]), lattice, Uint32Array.from([0, 1, 2, 3, 4]))
        expect([...terrain]).toEqual([kindOf("grass"), seaKind(1), seaKind(2), seaKind(1), kindOf("sand")])
    })
})

describe("kindsFromNeighbours", () => {
    it("gives a tile where earth-1 had sea the kind most of its land neighbours have", () => {
        const {lattice, points} = graphOf(5, [[0, 4], [1, 4], [2, 4], [3, 4]], [0, 1, 2, 3, 4])
        const kinds = Uint8Array.from([kindOf("forest"), kindOf("sand"), kindOf("sand"), kindOf("snow"), kindOf("tundra")])
        const astray = Uint8Array.from([0, 0, 0, 0, 1])
        expect(kindsFromNeighbours(kinds, astray, lattice, points, () => kindOf("grass"))).toEqual({neighbours: 1, fallback: 0})
        expect(kinds[4]).toBe(kindOf("sand"))
    })

    it("reaches a coast two tiles deep, wave by wave, and leaves the photo's land alone", () => {
        const {lattice, points} = line(4, [0, 1, 2, 3])
        const kinds = Uint8Array.from([kindOf("arid"), kindOf("snow"), kindOf("snow"), kindOf("forest")])
        kindsFromNeighbours(kinds, Uint8Array.from([0, 1, 1, 0]), lattice, points, () => kindOf("grass"))
        expect([...kinds]).toEqual([kindOf("arid"), kindOf("arid"), kindOf("forest"), kindOf("forest")])
    })

    it("gives an island no land neighbour reaches the fallback's kind", () => {
        const {lattice, points} = line(3, [0, 2])
        const kinds = Uint8Array.from([kindOf("arid"), kindOf("snow")])
        expect(kindsFromNeighbours(kinds, Uint8Array.from([0, 1]), lattice, points, (tile) => tile === 1 ? kindOf("savanna") : 0))
            .toEqual({neighbours: 0, fallback: 1})
        expect(kinds[1]).toBe(kindOf("savanna"))
    })
})
