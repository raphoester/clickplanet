import {readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {BOARD} from "../app/viewer/boardAsset.ts"
import {decodeRuns, tilesOf} from "./board.ts"
import {latticeIds, latticePositions, latticeSize} from "./lattice.ts"
import {cellOf, cellsOf, IDS_PER_WORD, latticeFaces, nearestOnGrid} from "./latticeCells.ts"

const STATIC = fileURLToPath(new URL("../../static/", import.meta.url))

const COLS = 13

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
    })
})

describe("cellOf", () => {
    const faces = latticeFaces(COLS)
    const positions = latticePositions(COLS)
    const ids = latticeIds(COLS)

    it("finds every lattice point in its own cell", () => {
        for (let point = 0; point < latticeSize(COLS); point++) {
            expect(cellOf(faces, COLS, ...pointOf(positions, point)), `point ${point}`).toBe(ids[point])
        }
    })

    it("keeps a point moved a third of the way to a neighbour in its cell", () => {
        const spacing = 1.1 / COLS
        for (let point = 0; point < latticeSize(COLS); point += 7) {
            const [x, y, z] = pointOf(positions, point)
            const side = Math.abs(x) < 0.9 ? [1, 0, 0] : [0, 1, 0]
            const across = [y * side[2] - z * side[1], z * side[0] - x * side[2], x * side[1] - y * side[0]]
            const length = Math.hypot(...across)
            const step = spacing / 3 / length
            const moved = [x + across[0] * step, y + across[1] * step, z + across[2] * step] as const
            expect(cellOf(faces, COLS, ...moved), `point ${point}`).toBe(ids[point])
        }
    })
})

describe("cellsOf", () => {
    const ids = latticeIds(COLS)
    const kept = Array.from({length: latticeSize(COLS)}, (_, point) => point).filter((point) => point % 3 !== 1)
    const tileIds = Uint32Array.from(kept, (point) => ids[point])

    it("gives every tile its own texel, in canonical order", () => {
        const {texels} = cellsOf(tileIds, COLS)
        const byId = kept.map((point, tile) => ({id: ids[point], tile})).sort((a, b) => a.id - b.id)
        byId.forEach(({tile}, rank) => expect(texels[tile]).toBe(rank))
    })

    it("marks the tiles' ids and counts the tiles before each word", () => {
        const {words, texels, cols} = cellsOf(tileIds, COLS)
        expect(cols).toBe(COLS)
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

    it("refuses two tiles in one cell", () => {
        expect(() => cellsOf(Uint32Array.from([4, 4]), COLS)).toThrow(/another tile/)
    })
})

describe("the shipped board", () => {
    it("puts each tile in its own cell as the territory layer finds it", () => {
        const bytes = readFileSync(`${STATIC}boards/${BOARD.name}/${BOARD.files.runs}`)
        const runs = decodeRuns(bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength))
        const {positions, ids} = tilesOf(BOARD.lattice, runs, BOARD.tiles)
        const faces = latticeFaces(BOARD.lattice.cols)

        for (let tile = 0; tile < BOARD.tiles; tile++) {
            const found = cellOf(faces, BOARD.lattice.cols, ...pointOf(positions, tile))
            if (found !== ids[tile]) expect(found, `tile ${tile + 1}`).toBe(ids[tile])
        }
        expect(() => cellsOf(ids, BOARD.lattice.cols)).not.toThrow()
    })
})

function popcount(bits: number): number {
    let count = 0
    for (let rest = bits >>> 0; rest; rest >>>= 1) count += rest & 1
    return count
}
