import {describe, expect, it} from "vitest"
import {BLAST_TIMELINE, blastOver, describeBlast, nearestTile, tilesWithin} from "./blast.ts"

// Four tiles on the equator a quarter turn apart, and one just beside the first.
const POSITIONS = new Float32Array([
    1, 0, 0,
    0, 1, 0,
    -1, 0, 0,
    0, -1, 0,
    Math.cos(0.05), Math.sin(0.05), 0,
])

describe("tilesWithin", () => {
    it("takes the centre and whatever is inside the radius", () => {
        expect(tilesWithin(POSITIONS, 1, 0.1)).toEqual([1, 5])
    })

    it("measures along the surface, not across the chord", () => {
        // A quarter turn is π/2 of arc; the chord would be √2.
        expect(tilesWithin(POSITIONS, 1, Math.PI / 2 - 0.01)).toEqual([1, 5])
        expect(tilesWithin(POSITIONS, 1, Math.PI / 2 + 0.01)).toEqual([1, 2, 4, 5])
    })

    it("does not depend on how far a position is from the origin", () => {
        const scaled = POSITIONS.map((v) => v * 3)
        expect(tilesWithin(scaled, 1, 0.1)).toEqual([1, 5])
    })

    it("clears nothing for a tile that is not on the map", () => {
        expect(tilesWithin(POSITIONS, 0, 1)).toEqual([])
        expect(tilesWithin(POSITIONS, 6, 1)).toEqual([])
    })
})

describe("nearestTile", () => {
    it("finds the closest tile and how far the aim is from it", () => {
        const {tile, arc, point} = nearestTile(POSITIONS, {x: 3 * Math.cos(0.03), y: 3 * Math.sin(0.03), z: 0})

        expect(tile).toBe(5)
        expect(arc).toBeCloseTo(0.02, 5)
        expect(Math.hypot(point.x, point.y, point.z)).toBeCloseTo(1, 9)
    })

    it("finds nothing for an aim with no direction", () => {
        expect(nearestTile(POSITIONS, {x: 0, y: 0, z: 0}).tile).toBeUndefined()
    })
})

describe("describeBlast", () => {
    it("says how much a bomb took, and owns up to a miss", () => {
        expect(describeBlast({tile: 7, cleared: [6, 7, 8]})).toBe("bombed 3 tiles")
        expect(describeBlast({tile: 7, cleared: [7]})).toBe("bombed 1 tile")
        expect(describeBlast({tile: 7, cleared: []})).toBe("bombed empty land")
        expect(describeBlast({tile: undefined, cleared: []})).toBe("bombed the ocean")
    })

    it("names the country whose ground was hit", () => {
        expect(describeBlast({tile: 7, cleared: [6, 7, 8]}, "Germany")).toBe("bombed Germany")
        expect(describeBlast({tile: 7, cleared: []}, "Germany")).toBe("bombed Germany")
        expect(describeBlast({tile: undefined, cleared: []}, "Germany")).toBe("bombed the ocean")
    })
})

describe("blastOver", () => {
    it("lasts through the fall and the scorch", () => {
        expect(blastOver(BLAST_TIMELINE.fall)).toBe(false)
        expect(blastOver(BLAST_TIMELINE.fall + BLAST_TIMELINE.scorch)).toBe(true)
    })
})
