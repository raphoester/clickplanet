import {describe, expect, it} from "vitest"
import {cellDegreesOf, HeatBoard, heatOf, placeOf} from "./heatmap.ts"
import {EARTH_1_SCALE, scaleOf} from "../../domain/boardScale.ts"
import {latticeFaces, tileAt} from "../../domain/latticeCells.ts"
import {type Board} from "../../domain/board.ts"
import {EARTH_1, FIXTURE_1, tilesOfBoard} from "../../testdata/boards.ts"

// The globe's frame, as the backoffice reads it: y is north, -x is longitude 0, z is 90° east.
const at = (x: number, y: number, z: number) => Float32Array.from([x, y, z])

describe("placeOf", () => {
    it("reads a tile's latitude and longitude off its position", () => {
        expect(placeOf(at(-1, 0, 0), 1)).toEqual({latitude: 0, longitude: 0})
        expect(placeOf(at(0, 0, 1), 1).longitude).toBeCloseTo(90)
        expect(placeOf(at(0, 1, 0), 1).latitude).toBeCloseTo(90)
        expect(Math.abs(placeOf(at(1, 0, 0), 1).longitude)).toBeCloseTo(180)
    })
})

describe("a heatmap's cell", () => {
    it("is 0.4° on earth-1, and as big as a coarser board's own tiles", () => {
        expect(cellDegreesOf(EARTH_1_SCALE)).toBe(0.4)
        expect(cellDegreesOf(scaleOf(FIXTURE_1))).toBeCloseTo(0.4 * 9.697, 2)
    })
})

describe("heatOf", () => {
    const boardOf = (board: Board): HeatBoard => {
        const {positions, cells} = tilesOfBoard(board)
        return {name: board.name, spacing: board.spacing, positions, cells}
    }
    const earth = boardOf(EARTH_1)
    const fixture = boardOf(FIXTURE_1)
    const pointOf = (board: HeatBoard, tile: number) => [board.positions[(tile - 1) * 3], board.positions[(tile - 1) * 3 + 1], board.positions[(tile - 1) * 3 + 2]] as const

    it("is the one era's hits on its own board", () => {
        const hits = new Map([[7, 2], [90_000, 5]])
        const heat = heatOf([{board: earth, hits}])

        expect(heat.board).toBe(earth)
        expect(heat.hits).toEqual(hits)
    })

    it("lays every era's hits on the finest board among them, each where it was made, and adds them up", () => {
        // A tile of earth-1, and the tile of fixture-1 its centre is on.
        const inland = 61_000
        const faces = latticeFaces(FIXTURE_1.lattice.cols)
        const under = tileAt(fixture.cells, faces, ...pointOf(earth, inland))
        expect(under).toBeDefined()

        for (const eras of [
            [{board: earth, hits: new Map([[inland, 2]])}, {board: fixture, hits: new Map([[under!, 3]])}],
            [{board: fixture, hits: new Map([[under!, 3]])}, {board: earth, hits: new Map([[inland, 2]])}],
        ]) {
            const heat = heatOf(eras)

            expect(heat.board).toBe(earth)
            expect(heat.hits.get(inland)).toBe(5)
            const covered = [...heat.hits.keys()].filter((tile) => tile !== inland)
            expect(covered.length).toBeGreaterThan(20)
            for (const tile of covered) {
                expect(heat.hits.get(tile)).toBe(3)
                expect(tileAt(fixture.cells, faces, ...pointOf(earth, tile))).toBe(under)
            }
        }
    })
})
