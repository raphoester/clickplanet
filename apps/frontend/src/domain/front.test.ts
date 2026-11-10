import {describe, expect, it} from "vitest"
import {Adjacency, Front} from "./front.ts"
import {Waters} from "./waters.ts"

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

function frontOf(owners: Map<number, string>, waters?: Waters): Front {
    return new Front(row, (tile) => owners.get(tile), positionsOfRow(), waters)
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

describe("Front across waters", () => {
    it("lights every coast of a water where the flag holds a coast, as well as its land's edge", () => {
        const front = frontOf(new Map([[1, "fr"]]), new Waters([[1, 6]]))

        expect(front.follow("fr").on.sort()).toEqual([2, 6])
    })

    it("lights a water's coasts when the flag takes its first coast there, and puts them out when it loses its last", () => {
        const owners = new Map([[2, "fr"]])
        const front = frontOf(owners, new Waters([[1, 6]]))
        front.follow("fr")

        owners.set(1, "fr")
        expect(front.touch([1])).toEqual({on: [6], off: [1]})

        owners.set(1, "de")
        expect(front.touch([1])).toEqual({on: [1], off: [6]})
    })

    it("counts a tile between two waters as a coast of both", () => {
        const owners = new Map([[3, "fr"]])
        const front = frontOf(owners, new Waters([[1, 6], [3, 6]]))
        expect(front.follow("fr").on.sort()).toEqual([2, 4, 6])

        owners.set(6, "fr")
        const {on, off} = front.touch([6])
        expect(on.sort()).toEqual([1, 5])
        expect(off).toEqual([6])
    })

    it("keeps a water lit while the flag holds another of its coasts", () => {
        const owners = new Map([[1, "fr"], [3, "fr"]])
        const front = frontOf(owners, new Waters([[1, 3, 6]]))
        front.follow("fr")

        owners.set(1, "de")
        front.touch([1])

        expect(front.reaches(6)).toBe(true)
        expect(front.reaches(1)).toBe(true)
    })

    it("reaches the flag's own tiles and the lit ones, and nothing else", () => {
        const front = frontOf(new Map([[1, "fr"]]), new Waters([[1, 6]]))
        expect(front.reaches(1)).toBe(false)

        front.follow("fr")

        expect([1, 2, 3, 4, 5, 6].filter((tile) => front.reaches(tile))).toEqual([1, 2, 6])
    })

    it("puts out every lit tile when dropped, and reaches nothing after", () => {
        const front = frontOf(new Map([[1, "fr"]]), new Waters([[1, 6]]))
        front.follow("fr")

        const {on, off} = front.drop()

        expect(on).toEqual([])
        expect(off.sort()).toEqual([2, 6])
        expect(front.reaches(2)).toBe(false)
    })

    it("faces a point when a lit tile is on its side of the globe", () => {
        const front = frontOf(new Map([[3, "fr"]]))
        front.follow("fr")

        expect(front.facing({x: 1, y: 0, z: 0})).toBe(true)
        expect(front.facing({x: -1, y: 0, z: 0})).toBe(false)
    })
})
