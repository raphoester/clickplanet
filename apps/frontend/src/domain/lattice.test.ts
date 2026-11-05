import {describe, expect, it} from "vitest"
import {latticePositions, latticeSize, positionsHash, tilePositions} from "./lattice.ts"

const everyOther = (points: number) => {
    const isTile = new Uint8Array(Math.ceil(points / 8))
    for (let point = 0; point < points; point += 2) isTile[point >> 3] |= 1 << (point & 7)
    return isTile
}

describe("latticePositions", () => {
    it("holds the geodesic sphere's 10*cols^2 + 2 points, each on the unit sphere", () => {
        const positions = latticePositions(4)
        expect(positions.length).toBe(latticeSize(4) * 3)
        for (let point = 0; point < latticeSize(4); point++) {
            const [x, y, z] = positions.subarray(point * 3, point * 3 + 3)
            expect(Math.sqrt(x * x + y * y + z * z)).toBeCloseTo(1, 6)
        }
    })

    it("holds each point once", () => {
        const positions = latticePositions(6)
        const keys = new Set<string>()
        for (let point = 0; point < latticeSize(6); point++) keys.add(positions.subarray(point * 3, point * 3 + 3).join(","))
        expect(keys.size).toBe(latticeSize(6))
    })
})

describe("tilePositions", () => {
    it("keeps the masked points, in lattice order", () => {
        const all = latticePositions(3)
        const kept = tilePositions(3, everyOther(latticeSize(3)), Math.ceil(latticeSize(3) / 2))

        for (let tile = 0; tile < kept.length / 3; tile++) {
            expect(Array.from(kept.subarray(tile * 3, tile * 3 + 3))).toEqual(Array.from(all.subarray(tile * 6, tile * 6 + 3)))
        }
    })

    it("refuses a mask that keeps another number of tiles than it says", () => {
        const isTile = everyOther(latticeSize(3))
        const kept = Math.ceil(latticeSize(3) / 2)
        expect(() => tilePositions(3, isTile, kept - 1)).toThrow(/more than/)
        expect(() => tilePositions(3, isTile, kept + 1)).toThrow(/expected/)
    })
})

describe("positionsHash", () => {
    it("changes when one bit of one coordinate does", () => {
        const positions = latticePositions(2)
        const before = positionsHash(positions)
        new Uint32Array(positions.buffer)[17] ^= 1
        expect(positionsHash(positions)).not.toBe(before)
    })
})
