import {readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {BOARDS} from "../app/viewer/boardsAsset.ts"
import {decodeRuns, tilesOf} from "./board.ts"
import {type Lattice, latticeIds, latticePositions, latticeSize, threeLattice} from "./lattice.ts"
import {cellOf, cellsOf, IDS_PER_WORD, latticeFaces, nearestOnGrid, tileAt} from "./latticeCells.ts"

const STATIC = fileURLToPath(new URL("../../static/", import.meta.url))

const COLS = 13

// A warped and rotated lattice as earth-2's and siege-1's, at a few cols: the cells the shader finds must be its own.
const NEW = BOARDS["earth-2"].lattice
const warpedAt = (cols: number): Lattice => ({...NEW, cols})

function pointOf(positions: Float32Array, point: number): [number, number, number] {
    return [positions[point * 3], positions[point * 3 + 1], positions[point * 3 + 2]]
}

describe("nearestOnGrid", () => {
    it("rounds to the corner with the largest weight", () => {
        expect(nearestOnGrid(0.2, 0.1)).toEqual([0, 0])
        expect(nearestOnGrid(0.7, 0.1)).toEqual([1, 0])
        expect(nearestOnGrid(0.1, 0.7)).toEqual([0, 1])
        expect(nearestOnGrid(0.9, 0.8)).toEqual([1, 1])
        expect(nearestOnGrid(3.8, 2.7)).toEqual([4, 3])
        expect(nearestOnGrid(-0.4, 0.1)).toEqual([0, 0])
        expect(nearestOnGrid(-0.7, 0.1)).toEqual([-1, 0])
    })
})

describe("cellOf", () => {
    for (const lattice of [threeLattice(COLS), warpedAt(COLS), warpedAt(40)]) {
        const name = lattice.warp === null ? `three's lattice at ${lattice.cols}` : `a warped, rotated lattice at ${lattice.cols}`
        const faces = latticeFaces(lattice)
        const positions = latticePositions(lattice)
        const ids = latticeIds(lattice)
        const spacing = 1.1 / lattice.cols

        it(`finds every lattice point in its own cell, on ${name}`, () => {
            for (let point = 0; point < latticeSize(lattice.cols); point++) {
                const found = cellOf(faces, lattice, ...pointOf(positions, point))
                if (found !== ids[point]) expect(found, `point ${point}`).toBe(ids[point])
            }
        })

        it(`keeps a point moved a third of the way to a neighbour in its cell, on ${name}`, () => {
            for (let point = 0; point < latticeSize(lattice.cols); point += 7) {
                const [x, y, z] = pointOf(positions, point)
                const side = Math.abs(x) < 0.9 ? [1, 0, 0] : [0, 1, 0]
                const across = [y * side[2] - z * side[1], z * side[0] - x * side[2], x * side[1] - y * side[0]]
                const step = spacing / 3 / Math.hypot(...across)
                const moved = [x + across[0] * step, y + across[1] * step, z + across[2] * step] as const
                expect(cellOf(faces, lattice, ...moved), `point ${point}`).toBe(ids[point])
            }
        })
    }
})

describe("cellsOf", () => {
    const ids = latticeIds(COLS)
    const positions = latticePositions(COLS)
    const kept = Array.from({length: latticeSize(COLS)}, (_, point) => point).filter((point) => point % 3 !== 1)
    const tileIds = Uint32Array.from(kept, (point) => ids[point])
    const tilePositions = Float32Array.from(kept.flatMap((point) => pointOf(positions, point)))

    it("gives every tile its own texel, in canonical order", () => {
        const {texels} = cellsOf(tileIds, COLS, tilePositions)
        const byId = kept.map((point, tile) => ({id: ids[point], tile})).sort((a, b) => a.id - b.id)
        byId.forEach(({tile}, rank) => expect(texels[tile]).toBe(rank))
    })

    it("marks the tiles' ids and counts the tiles before each word", () => {
        const {words, texels, lattice} = cellsOf(tileIds, COLS, tilePositions)
        expect(lattice).toEqual(threeLattice(COLS))
        kept.forEach((point, tile) => {
            const id = ids[point]
            const word = Math.floor(id / IDS_PER_WORD)
            const bit = id % IDS_PER_WORD
            const bits = words[word * 2]
            expect((bits >>> bit) & 1).toBe(1)
            const below = bits & ((1 << bit) - 1)
            expect(words[word * 2 + 1] + popcount(below)).toBe(texels[tile])
        })
    })

    it("names the tile at each texel", () => {
        const {texels, tiles} = cellsOf(tileIds, COLS, tilePositions)
        texels.forEach((texel, tile) => expect(tiles[texel]).toBe(tile))
    })

    it("finds each tile at its own position, and nothing in a cell that is not a tile", () => {
        const cells = cellsOf(tileIds, COLS, tilePositions)
        const faces = latticeFaces(COLS)
        kept.forEach((point, tile) => expect(tileAt(cells, faces, ...pointOf(positions, point))).toBe(tile + 1))
        const sea = Array.from({length: latticeSize(COLS)}, (_, point) => point).filter((point) => point % 3 === 1)
        for (const point of sea) expect(tileAt(cells, faces, ...pointOf(positions, point)), `point ${point}`).toBeUndefined()
    })

    it("refuses two tiles in one cell", () => {
        const four = pointOf(positions, ids.indexOf(4))
        expect(() => cellsOf(Uint32Array.from([4, 4]), COLS, Float32Array.from([...four, ...four]))).toThrow(/another tile/)
    })

    it("refuses a tile whose position the inverse finds in another cell: the globe does not load", () => {
        const astray = Float32Array.from(tilePositions)
        astray.set(pointOf(positions, kept[1]), 0)
        expect(() => cellsOf(tileIds, COLS, astray)).toThrow(/tile 1, point \d+, is found in the cell of point \d+/)
    })

    it("refuses a warped board read without its inverse: the warp moves a point by more than half a cell at 100", () => {
        const lattice = warpedAt(100)
        const every = Uint32Array.from({length: latticeSize(100)}, (_, id) => id)
        const warped = latticePositions(lattice)
        expect(() => cellsOf(every, {...lattice, warp: null}, warped)).toThrow(/is found in the cell of/)
        expect(() => cellsOf(every, lattice, warped)).not.toThrow()
    })
})

describe("the shipped boards", () => {
    for (const board of Object.values(BOARDS)) {
        it(`put each tile of ${board.name} in its own cell as the territory layer finds it`, () => {
            const bytes = readFileSync(`${STATIC}boards/${board.name}/${board.files.runs}`)
            const runs = decodeRuns(bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength))
            const {positions, ids} = tilesOf(board.lattice, runs, board.tiles)
            const faces = latticeFaces(board.lattice)

            let astray = 0
            for (let tile = 0; tile < board.tiles; tile++) {
                if (cellOf(faces, board.lattice, ...pointOf(positions, tile)) !== ids[tile]) astray++
            }
            expect(astray).toBe(0)
            const cells = cellsOf(ids, board.lattice, positions)
            for (let tile = 0; tile < board.tiles; tile += 97) {
                expect(tileAt(cells, faces, ...pointOf(positions, tile)), `tile ${tile + 1}`).toBe(tile + 1)
            }
        }, 60_000)
    }
})

function popcount(bits: number): number {
    let count = 0
    for (let rest = bits >>> 0; rest; rest >>>= 1) count += rest & 1
    return count
}
