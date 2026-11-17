import {describe, expect, it} from "vitest"
import * as THREE from "three"

import {latticeTriangles} from "../../src/domain/lattice.ts"
import {EARTH_1} from "./board.mjs"
import {keyOf, lattice, lonLatOf, neighbours} from "./lattice.mjs"

// The generator walks the lattice's own arithmetic; three is only the reference it is checked against.
function three(cols) {
    const geometry = new THREE.IcosahedronGeometry(1, cols - 1)
    const position = geometry.attributes.position.array
    const uv = geometry.attributes.uv.array
    geometry.dispose()

    const pointOf = new Map()
    const positions = []
    const uvs = []
    const corners = new Uint32Array(position.length / 3)
    for (let i = 0; i < position.length; i += 3) {
        const key = keyOf(position[i], position[i + 1], position[i + 2])
        let point = pointOf.get(key)
        if (point === undefined) {
            pointOf.set(key, point = pointOf.size)
            positions.push(position[i], position[i + 1], position[i + 2])
            uvs.push(uv[(i / 3) * 2], uv[(i / 3) * 2 + 1])
        }
        corners[i / 3] = point
    }
    return {positions: Float32Array.from(positions), uvs, corners}
}

// Each edge of three's triangles once, as smaller * count + larger.
function edgesOf(corners, count) {
    const edges = new Set()
    for (let t = 0; t < corners.length; t += 3) {
        for (const [a, b] of [[0, 1], [1, 2], [0, 2]]) {
            const [p, q] = [corners[t + a], corners[t + b]]
            edges.add(Math.min(p, q) * count + Math.max(p, q))
        }
    }
    return edges
}

describe("lattice", () => {
    it("holds 10*cols^2 + 2 points, the geodesic sphere's own count", () => {
        for (const cols of [2, 5, 11]) expect(lattice(cols).count).toBe(10 * cols * cols + 2)
    })

    it("is three's IcosahedronGeometry, each point once, in three's order and to the bit", () => {
        for (const cols of [2, 5, 11, EARTH_1.lattice.cols]) {
            const want = new Uint32Array(three(cols).positions.buffer)
            const got = new Uint32Array(lattice(cols).positions.buffer)
            expect(got.length).toBe(want.length)
            let differing = 0
            for (let k = 0; k < want.length; k++) if (got[k] !== want[k]) differing++
            expect(differing, `cols ${cols}`).toBe(0)
        }
    }, 120_000)

    it("walks three's triangles, in three's order", () => {
        for (const cols of [2, 5, 11]) expect(Array.from(latticeTriangles(cols))).toEqual(Array.from(three(cols).corners))
    })
})

describe("neighbours", () => {
    it("are the edges of three's triangles, each listed from both ends, ascending", () => {
        for (const cols of [2, 5, 11, EARTH_1.lattice.cols]) {
            const {count, at, to} = neighbours(cols)
            const want = edgesOf(three(cols).corners, count)
            let listed = 0, astray = 0, unsorted = 0, oneWay = 0
            for (let p = 0; p < count; p++) {
                for (let e = at[p]; e < at[p + 1]; e++) {
                    const q = to[e]
                    if (e > at[p] && to[e - 1] >= q) unsorted++
                    if (!to.subarray(at[q], at[q + 1]).includes(p)) oneWay++
                    if (!want.has(Math.min(p, q) * count + Math.max(p, q))) astray++
                    listed++
                }
            }
            expect({listed, astray, unsorted, oneWay}, `cols ${cols}`).toEqual({listed: want.size * 2, astray: 0, unsorted: 0, oneWay: 0})
        }
    }, 120_000)

    it("gives every point 6 neighbours and the 12 corners 5", () => {
        const {count, at} = neighbours(11)
        const degrees = new Map()
        for (let p = 0; p < count; p++) degrees.set(at[p + 1] - at[p], (degrees.get(at[p + 1] - at[p]) ?? 0) + 1)
        expect(Object.fromEntries(degrees)).toEqual({5: 12, 6: count - 12})
    })
})

describe("lonLatOf", () => {
    it("is the equirectangular projection three's uvs make of the same point", () => {
        const cols = 11
        const {positions} = lattice(cols)
        const {uvs} = three(cols)
        let worst = 0
        for (let v = 0; v < positions.length / 3; v++) {
            const [lon, lat] = lonLatOf(positions, v)
            const [u, w] = [uvs[v * 2], uvs[v * 2 + 1]]
            let dLon = Math.abs(((u - 0.5) * 360 - lon + 540) % 360 - 180)
            if (Math.abs(lat) > 89.9) dLon = 0
            worst = Math.max(worst, dLon * Math.cos(lat * Math.PI / 180), Math.abs((w - 0.5) * 180 - lat))
        }
        expect(worst).toBeLessThan(0.01)
    })
})
