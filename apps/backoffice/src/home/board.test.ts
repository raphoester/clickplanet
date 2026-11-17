import {readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {BOARD} from "../gen/map/board.ts"
import {boardPositions, positionsHash} from "./board.ts"
import {boardPlaces} from "./places.ts"

const GEN = fileURLToPath(new URL("../gen/map/", import.meta.url))
const MAP = fileURLToPath(new URL("../../../../map/", import.meta.url))

function runs(): ArrayBuffer {
    const bytes = readFileSync(`${GEN}${BOARD.name}/${BOARD.files.runs}`)
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)
}

describe("the bundled board", () => {
    it("rebuilds the positions its manifest hashes", () => {
        const positions = boardPositions(BOARD.lattice, BOARD.tiles, runs())
        expect(positions.length).toBe(BOARD.tiles * 3)
        expect(positionsHash(positions)).toBe(BOARD.positionsHash)
    })

    it("is /map's", () => {
        const directory = `${MAP}boards/${BOARD.name}/`
        expect(readFileSync(`${GEN}${BOARD.name}/${BOARD.files.runs}`).equals(readFileSync(directory + BOARD.files.runs))).toBe(true)
    })

    it("puts tile 1 where the game does", () => {
        const places = boardPlaces(runs())
        expect(places.latitudes.length).toBe(BOARD.tiles)
        expect(places.latitudes[0]).toBeCloseTo(Math.asin(0.8018954992294312) * 180 / Math.PI, 4)
        expect(places.longitudes[0]).toBeCloseTo(Math.atan2(0.06102520972490311, 0.5943395495414734) * 180 / Math.PI, 4)
    })
})

describe("boardPositions", () => {
    it("refuses a file that is not runs", () => {
        expect(() => boardPositions(BOARD.lattice, 1, new ArrayBuffer(12))).toThrow("not a runs file")
    })

    it("refuses runs that hold another number of tiles than the board says", () => {
        expect(() => boardPositions(BOARD.lattice, BOARD.tiles - 1, runs())).toThrow("more than")
        expect(() => boardPositions(BOARD.lattice, BOARD.tiles + 1, runs())).toThrow("the board says")
    })
})
