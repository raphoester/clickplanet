// The dedup rule, at a detail small enough to run in the suite. What it has to keep is the rule
// itself — 6-decimal key, first uv wins, generation order — because that is what makes a position
// in an existing blob still match a lattice vertex, and so what keeps tile ids where they are.
//
// The whole lattice at DETAIL is 906,012 vertices and needs a raised heap; `npm run map:audit`
// checks it against the shipped blob, and the generator refuses to write a blob that does not.
import {describe, expect, it} from "vitest"
import * as THREE from "three"

import {DETAIL, keyOf, lattice, lonLatOf} from "./lattice.mjs"

describe("lattice", () => {
    it("holds 10*cols^2 + 2 vertices, the geodesic sphere's own count", () => {
        for (const detail of [1, 4, 10]) {
            const cols = detail + 1
            expect(lattice(detail).count).toBe(10 * cols * cols + 2)
        }
    })

    it("keeps every position once", () => {
        const {count, positions} = lattice(6)
        const seen = new Set()
        for (let i = 0; i < count; i++) {
            seen.add(keyOf(positions[i * 3], positions[i * 3 + 1], positions[i * 3 + 2]))
        }
        expect(seen.size).toBe(count)
    })

    it("keeps the first uv a position was generated with, in generation order", () => {
        const detail = 6
        const geometry = new THREE.IcosahedronGeometry(1, detail)
        const position = geometry.attributes.position.array
        const uv = geometry.attributes.uv.array

        const expected = []
        const seen = new Set()
        for (let i = 0; i < position.length; i += 3) {
            const key = keyOf(position[i], position[i + 1], position[i + 2])
            if (seen.has(key)) continue
            seen.add(key)
            expected.push(key, uv[(i / 3) * 2], uv[(i / 3) * 2 + 1])
        }

        const {count, positions, uvs} = lattice(detail)
        const actual = []
        for (let i = 0; i < count; i++) {
            actual.push(keyOf(positions[i * 3], positions[i * 3 + 1], positions[i * 3 + 2]), uvs[i * 2], uvs[i * 2 + 1])
        }
        expect(actual).toEqual(expected)
    })

    // The uvs are what the land mask, the borders and the globe's texture are all sampled with, so
    // a uv that has drifted from its own position moves a tile into the wrong country. three lays
    // them out equirectangularly; this says so in a way that fails if that ever changes.
    it("carries uvs that are the equirectangular projection of their own positions", () => {
        const {count, positions, uvs} = lattice(10)
        let worst = 0
        for (let i = 0; i < count; i++) {
            const [x, y, z] = [positions[i * 3], positions[i * 3 + 1], positions[i * 3 + 2]]
            const [lon, lat] = lonLatOf(uvs[i * 2], uvs[i * 2 + 1])
            const fromPosition = [
                Math.atan2(z, -x) * 180 / Math.PI,
                Math.asin(Math.max(-1, Math.min(1, y))) * 180 / Math.PI,
            ]
            let dLon = Math.abs(lon - fromPosition[0])
            if (dLon > 180) dLon = 360 - dLon
            // A pole's longitude is undefined, and the seam's two uvs both name the same meridian.
            if (Math.abs(fromPosition[1]) > 89.9) dLon = 0
            if (Math.abs(dLon - 360) < 1e-3 || Math.abs(dLon) > 179.9) dLon = 0
            worst = Math.max(worst, dLon * Math.cos(fromPosition[1] * Math.PI / 180), Math.abs(lat - fromPosition[1]))
        }
        expect(worst).toBeLessThan(0.01)
    })

    it("defaults to the detail the shipped map was cut at", () => {
        expect(DETAIL).toBe(300)
    })
})
