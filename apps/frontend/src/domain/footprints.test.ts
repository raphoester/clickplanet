import {describe, expect, it} from "vitest"
import {eastOf, Footprints, NO_FOOTPRINT, unitPositions} from "./footprints.ts"

// Tiles on the equator at these longitudes (degrees), plus one at (lat, lon) when given as a pair.
function tilesAt(points: ([number, number] | number)[]): Float32Array {
    const out = new Float32Array(points.length * 3)
    points.forEach((point, i) => {
        const [lat, lon] = typeof point === "number" ? [0, point] : point
        const φ = lat * Math.PI / 180, λ = lon * Math.PI / 180
        // y up, as the globe: x and z span the equator.
        out.set([Math.cos(φ) * Math.sin(λ), Math.sin(φ), Math.cos(φ) * Math.cos(λ)], i * 3)
    })
    return out
}

const FLAGS = 4

describe("Footprints", () => {
    it("gives each flag on each ground its own footprint, and the same one to its tiles", () => {
        const unit = tilesAt([0, 1, 2, 3])
        const f = new Footprints(unit, Uint16Array.from([1, 1, 2, 2]), 3, FLAGS, 16)
        const a = f.set(0, 1)
        expect(f.set(1, 1)).toBe(a)
        const b = f.set(2, 1)
        expect(b).not.toBe(a)
        expect(f.set(3, 2)).not.toBe(b)
        expect(f.countOf(a)).toBe(2)
    })

    it("centres the frame on the flag's own tiles, not on the ground's", () => {
        // One ground over 0..40°; the flag holds 30..40° alone.
        const f = new Footprints(tilesAt([30, 32, 34, 36, 38, 40]), new Uint16Array(6).fill(1), 2, FLAGS, 16)
        for (let tile = 0; tile < 6; tile++) f.set(tile, 3)
        const frame = f.frameOf(f.ofTile[0])!
        const lon = Math.atan2(frame.centre[0], frame.centre[2]) * 180 / Math.PI
        expect(lon).toBeCloseTo(35, 5)
        // An even spread of 10° east to west: about 5° each way, nothing north to south.
        expect(frame.halfEast * 180 / Math.PI).toBeGreaterThan(4.5)
        expect(frame.halfEast * 180 / Math.PI).toBeLessThan(6.5)
        expect(frame.halfNorth).toBeLessThan(1e-6)
    })

    it("moves the frame as tiles change hands, and frees an emptied footprint", () => {
        const f = new Footprints(tilesAt([0, 10, 20]), new Uint16Array(3).fill(1), 2, FLAGS, 16)
        const id = f.set(0, 1)
        f.set(1, 1)
        f.set(2, 1)
        f.set(0, 2)
        const lon = (frame = f.frameOf(id)!) => Math.atan2(frame.centre[0], frame.centre[2]) * 180 / Math.PI
        expect(lon()).toBeCloseTo(15, 5)
        f.set(1, 0)
        f.set(2, 0)
        expect(f.frameOf(id)).toBeUndefined()
        expect(f.countOf(id)).toBe(0)
        expect(f.takeChanged()).toContain(id)
        // Its number serves another flag.
        expect(f.set(1, 3)).toBe(id)
    })

    it("gives a single tile a frame of no extent: the shader's least radius covers it", () => {
        const f = new Footprints(tilesAt([[45, 10]]), Uint16Array.of(1), 2, FLAGS, 16)
        const frame = f.frameOf(f.set(0, 1))!
        expect(frame.halfEast).toBeCloseTo(0, 6)
        expect(frame.halfNorth).toBeCloseTo(0, 6)
    })

    it("leaves a tile without a footprint when the table is full, or unowned, or on no ground", () => {
        const f = new Footprints(tilesAt([0, 1, 2]), Uint16Array.from([1, 1, 0]), 2, FLAGS, 2)
        expect(f.set(0, 1)).not.toBe(NO_FOOTPRINT)
        expect(f.set(1, 2)).toBe(NO_FOOTPRINT)
        expect(f.set(2, 1)).toBe(NO_FOOTPRINT)
        expect(f.set(0, 0)).toBe(NO_FOOTPRINT)
    })

    it("visits each footprint with a tile once, with its ground and flag, and not an emptied one", () => {
        const f = new Footprints(tilesAt([0, 1, 2, 3]), Uint16Array.from([1, 1, 2, 2]), 3, FLAGS, 16)
        const a = f.set(0, 1)
        f.set(1, 1)
        const b = f.set(2, 3)
        f.set(3, 2)
        f.set(3, 0)
        const seen: number[][] = []
        f.forEachFootprint((id, ground, flag) => seen.push([id, ground, flag]))
        expect(seen).toEqual([[a, 1, 1], [b, 2, 3]])
    })

    it("orients east as the shader does, any east at a pole", () => {
        expect(eastOf([0, 0, 1])).toEqual([1, 0, -0])
        expect(eastOf([0, 1, 0])).toEqual([1, 0, 0])
    })

    it("puts positions on the unit sphere", () => {
        const unit = unitPositions(Float32Array.of(0, 0, 2, 3, 4, 0))
        expect(Array.from(unit)).toEqual([0, 0, 1, 0.6000000238418579, 0.800000011920929, 0])
    })
})
