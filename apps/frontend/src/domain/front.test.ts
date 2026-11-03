import {describe, expect, it} from "vitest"
import {Adjacency, Front} from "./front.ts"

// Tiles 1 to 6 in a row along the equator, each touching the one before and the one after.
const ROW = 6

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

function frontOf(owners: Map<number, string>): Front {
    return new Front(row, (tile) => owners.get(tile), positionsOfRow())
}

describe("Front", () => {
    it("lights the tiles touching the flag's land that the flag does not hold, enemy or free", () => {
        const owners = new Map([[2, "de"], [3, "fr"], [4, "fr"], [5, "de"]])

        const {on, off} = frontOf(owners).follow("fr")

        expect(on.sort()).toEqual([2, 5])
        expect(off).toEqual([])
    })

    it("lights nothing for a flag that holds no tile", () => {
        expect(frontOf(new Map([[2, "de"]])).follow("fr").on).toEqual([])
    })

    it("moves the front when the flag takes a tile on it", () => {
        const owners = new Map([[3, "fr"]])
        const front = frontOf(owners)
        front.follow("fr")

        owners.set(4, "fr")
        const {on, off} = front.touch([4])

        expect(on).toEqual([5])
        expect(off).toEqual([4])
    })

    it("puts out the front of land the flag lost", () => {
        const owners = new Map([[3, "fr"]])
        const front = frontOf(owners)
        front.follow("fr")

        owners.set(3, "de")
        const {on, off} = front.touch([3])

        expect(on).toEqual([])
        expect(off.sort()).toEqual([2, 4])
    })

    it("hands back what to put out and to light when the player changes flag", () => {
        const owners = new Map([[2, "fr"], [5, "de"]])
        const front = frontOf(owners)
        front.follow("fr")

        const {on, off} = front.follow("de")

        expect(on.sort()).toEqual([4, 6])
        expect(off.sort()).toEqual([1, 3])
    })

    it("follows nothing before a flag is given", () => {
        expect(frontOf(new Map([[3, "fr"]])).touch([3])).toEqual({on: [], off: []})
    })

    it("finds the lit tile nearest a point", () => {
        const owners = new Map([[3, "fr"]])
        const front = frontOf(owners)
        front.follow("fr")

        expect(front.nearestTo({x: Math.cos(0.6), y: 0, z: Math.sin(0.6)})).toBe(4)
        expect(front.nearestTo({x: Math.cos(0.1), y: 0, z: Math.sin(0.1)})).toBe(2)
    })
})
