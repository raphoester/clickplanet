import {describe, expect, it} from "vitest"
import {latticeIds, latticeNeighbours, latticePositions, latticeSize, latticeTriangles, positionsHash} from "./lattice.ts"

describe("latticePositions", () => {
    it("holds the geodesic sphere's 10*cols^2 + 2 points, each on the unit sphere", () => {
        const positions = latticePositions(5)
        expect(positions.length).toBe(latticeSize(5) * 3)
        for (let point = 0; point < latticeSize(5); point++) {
            const [x, y, z] = positions.subarray(point * 3, point * 3 + 3)
            expect(Math.sqrt(x * x + y * y + z * z)).toBeCloseTo(1, 6)
        }
    })

    it("holds each point once", () => {
        const positions = latticePositions(7)
        const keys = new Set<string>()
        for (let point = 0; point < latticeSize(7); point++) keys.add(positions.subarray(point * 3, point * 3 + 3).join(","))
        expect(keys.size).toBe(latticeSize(7))
    })
})

describe("latticeIds", () => {
    it("gives each point its own canonical id, the twelve corners the first twelve", () => {
        const ids = latticeIds(7)
        const {at} = latticeNeighbours(7)
        expect(new Set(ids).size).toBe(latticeSize(7))
        expect(Math.max(...ids)).toBe(latticeSize(7) - 1)
        const corners = Array.from(ids).filter((_, point) => at[point + 1] - at[point] === 5)
        expect(corners.sort((a, b) => a - b)).toEqual(Array.from({length: 12}, (_, id) => id))
    })
})

describe("latticeNeighbours", () => {
    it("gives every point 6 neighbours and the 12 corners 5, each listed from both ends", () => {
        const cols = 6
        const {at, to} = latticeNeighbours(cols)
        let corners = 0
        for (let point = 0; point < latticeSize(cols); point++) {
            const row = to.subarray(at[point], at[point + 1])
            if (row.length === 5) corners++
            else expect(row.length).toBe(6)
            for (const other of row) expect(to.subarray(at[other], at[other + 1])).toContain(point)
        }
        expect(corners).toBe(12)
    })
})

describe("latticeTriangles", () => {
    it("covers the sphere with 20*cols^2 triangles of three neighbours each", () => {
        const cols = 4
        const corners = latticeTriangles(cols)
        const {at, to} = latticeNeighbours(cols)
        const touch = (a: number, b: number) => to.subarray(at[a], at[a + 1]).includes(b)
        expect(corners.length).toBe(20 * cols * cols * 3)
        for (let t = 0; t < corners.length; t += 3) {
            expect(touch(corners[t], corners[t + 1]) && touch(corners[t + 1], corners[t + 2]) && touch(corners[t], corners[t + 2])).toBe(true)
        }
    })
})

describe("positionsHash", () => {
    it("changes when one bit of one coordinate does", () => {
        const positions = latticePositions(3)
        const before = positionsHash(positions)
        new Uint32Array(positions.buffer)[17] ^= 1
        expect(positionsHash(positions)).not.toBe(before)
    })
})
