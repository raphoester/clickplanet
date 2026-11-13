import {describe, expect, it} from "vitest"
import {Adjacency, Front} from "./front.ts"
import {OpenGround, openTileNear, opensOn} from "./openTiles.ts"

// Tiles 1 to 8 in a row along the equator, each touching the one before and the one after.
const ROW = 8

const row: Adjacency = {
    of: (tile) => [tile - 1, tile + 1].filter((next) => next >= 1 && next <= ROW),
}

function positionsOfRow(): Float32Array {
    const positions = new Float32Array(ROW * 3)
    for (let tile = 1; tile <= ROW; tile++) {
        const angle = tile * 0.1
        positions.set([Math.cos(angle), 0, Math.sin(angle)], (tile - 1) * 3)
    }
    return positions
}

const at = (tile: number) => ({x: Math.cos(tile * 0.1), y: 0, z: Math.sin(tile * 0.1)})

function groundOf(owners: Map<number, string>, shields: Map<number, number>, reach = false): OpenGround & {front: Front} {
    const positions = positionsOfRow()
    const ownerOf = (tile: number) => owners.get(tile)
    const front = new Front(row, ownerOf, positions)
    front.follow("fr")
    return {flag: "fr", positions, ownerOf, shieldsOf: (tile) => shields.get(tile) ?? 0, reach, front}
}

describe("opensOn", () => {
    it("opens another flag's tile and a free one with no shield, never a shielded one or the flag's own", () => {
        const open = opensOn(groundOf(new Map([[1, "fr"], [2, "de"], [3, "de"]]), new Map([[3, 1]])))

        expect([1, 2, 3, 4].map(open)).toEqual([false, true, false, true])
    })
})

describe("openTileNear", () => {
    it("skips the shielded front for the nearest tile a click takes", () => {
        const ground = groundOf(new Map([[4, "fr"], [5, "fr"], [3, "de"], [6, "de"]]), new Map([[3, 2]]))

        expect(openTileNear(ground.front, at(3), ground)).toBe(6)
    })

    it("finds nothing when every front tile is shielded, so the flight keeps its old target", () => {
        const ground = groundOf(new Map([[4, "fr"], [3, "de"], [5, "de"]]), new Map([[3, 1], [5, 3]]))

        expect(openTileNear(ground.front, at(4), ground)).toBeUndefined()
        expect(ground.front.nearestTo(at(4))).toBeDefined()
    })

    it("finds the open tile nearest the country of a flag that holds nothing", () => {
        const ground = groundOf(new Map([[1, "de"], [2, "de"], [3, "de"], [7, "de"]]), new Map([[2, 1], [3, 1]]))

        expect(openTileNear(ground.front, at(2), ground)).toBe(1)
        expect(openTileNear(ground.front, at(3), ground)).toBe(4)
    })

    it("finds nothing for a flag that holds nothing while the reach stops a click away from its land", () => {
        const ground = groundOf(new Map([[1, "de"]]), new Map(), true)

        expect(openTileNear(ground.front, at(2), ground)).toBeUndefined()
    })
})
