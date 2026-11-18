import {readdirSync, readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {BOARDS, boardOf} from "./boards.ts"
import {boardPositions, positionsHash} from "./lattice.ts"
import {boardPlaces} from "./places.ts"
import {EARTH_1, FIXTURE_1, runsOf} from "./testBoards.ts"

const GEN = fileURLToPath(new URL("../gen/map/", import.meta.url))
const MAP = fileURLToPath(new URL("../../../../map/boards/", import.meta.url))

describe("the bundled boards", () => {
    it("are every board in /map, by name, each copied byte for byte", () => {
        const inMap = readdirSync(MAP, {withFileTypes: true}).filter(entry => entry.isDirectory()).map(entry => entry.name)
        expect([...BOARDS.keys()].sort()).toEqual(inMap.sort())

        for (const [name, {board}] of BOARDS) {
            expect(board.name).toBe(name)
            const manifest = readdirSync(`${MAP}${name}/`).find(file => /^board-[0-9a-f]{8}\.json$/.test(file))!
            expect(JSON.parse(readFileSync(`${MAP}${name}/${manifest}`, "utf8"))).toEqual(board)
            for (const file of [manifest, board.files.runs]) {
                expect(readFileSync(`${GEN}${name}/${file}`).equals(readFileSync(`${MAP}${name}/${file}`))).toBe(true)
            }
        }
    })

    it("each rebuild the positions their manifest hashes", () => {
        for (const {board} of BOARDS.values()) {
            const positions = boardPositions(board.lattice, board.tiles, runsOf(board))
            expect(positions.length).toBe(board.tiles * 3)
            expect(positionsHash(positions)).toBe(board.positionsHash)
        }
    })

    it("put earth-1's tile 1 where the game does", () => {
        const places = boardPlaces(EARTH_1, runsOf(EARTH_1))
        expect(places.board).toBe(EARTH_1)
        expect(places.latitudes.length).toBe(260503)
        expect(places.latitudes[0]).toBeCloseTo(Math.asin(0.8018954992294312) * 180 / Math.PI, 4)
        expect(places.longitudes[0]).toBeCloseTo(Math.atan2(0.06102520972490311, 0.5943395495414734) * 180 / Math.PI, 4)
    })
})

describe("boardOf", () => {
    it("finds a board by the name an answer gives", () => {
        expect(boardOf("earth-1").board).toEqual(EARTH_1)
        expect(boardOf("earth-1").runsUrl).toMatch(/runs-[0-9a-f]{8}\.bin/)
    })

    it("refuses a name the page does not carry, and an answer that names none", () => {
        expect(() => boardOf("fixture-1")).toThrow("fixture-1 is not a board this page carries (it has earth-1)")
        expect(() => boardOf("constructor")).toThrow("not a board this page carries")
        expect(() => boardOf("")).toThrow("the answer names no board")
    })
})

describe("a second board", () => {
    it("rebuilds fixture-1, the tests' coarse earth, to the positions its manifest hashes", () => {
        const positions = boardPositions(FIXTURE_1.lattice, FIXTURE_1.tiles, runsOf(FIXTURE_1))
        expect(FIXTURE_1.tiles).toBe(2998)
        expect(positionsHash(positions)).toBe(0xcb8330a8)
        expect(positionsHash(positions)).toBe(FIXTURE_1.positionsHash)
    })
})

describe("boardPositions", () => {
    it("refuses a file that is not runs", () => {
        expect(() => boardPositions(EARTH_1.lattice, 1, new ArrayBuffer(12))).toThrow("not a runs file")
    })

    it("refuses runs that hold another number of tiles than the board says", () => {
        expect(() => boardPositions(EARTH_1.lattice, EARTH_1.tiles - 1, runsOf(EARTH_1))).toThrow("more than")
        expect(() => boardPositions(EARTH_1.lattice, EARTH_1.tiles + 1, runsOf(EARTH_1))).toThrow("the board says")
    })

    it("refuses one board's runs on another's lattice", () => {
        expect(() => boardPositions(FIXTURE_1.lattice, FIXTURE_1.tiles, runsOf(EARTH_1))).toThrow("past the lattice")
    })
})
