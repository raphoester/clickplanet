import {readdirSync, readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {LATTICE_DETAIL, latticeIds, latticePositions, latticeSize} from "./lattice.ts"
import {cellOf, cellsOf, IDS_PER_WORD, LATTICE_COLS, latticeFaces, nearestOnGrid} from "./latticeCells.ts"
import {decodeLandmask, rebuildTilePositions} from "./landmask.ts"

const STATIC = fileURLToPath(new URL("../../static/", import.meta.url))

const DETAIL = 12
const COLS = DETAIL + 1

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
    const faces = latticeFaces(DETAIL)
    const positions = latticePositions(DETAIL)
    const ids = latticeIds(DETAIL)

    it("finds every lattice point in its own cell", () => {
        for (let point = 0; point < latticeSize(DETAIL); point++) {
            expect(cellOf(faces, COLS, ...pointOf(positions, point)), `point ${point}`).toBe(ids[point])
        }
    })

    it("keeps a point moved a third of the way to a neighbour in its cell", () => {
        const spacing = 1.1 / COLS
        for (let point = 0; point < latticeSize(DETAIL); point += 7) {
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
    const positions = latticePositions(DETAIL)
    const ids = latticeIds(DETAIL)
    const kept = Array.from({length: latticeSize(DETAIL)}, (_, point) => point).filter((point) => point % 3 !== 1)
    const tiles = new Float32Array(kept.flatMap((point) => pointOf(positions, point)))

    it("gives every tile its own texel, in canonical order", () => {
        const {texels} = cellsOf(tiles, DETAIL)
        const byId = kept.map((point, tile) => ({id: ids[point], tile})).sort((a, b) => a.id - b.id)
        byId.forEach(({tile}, rank) => expect(texels[tile]).toBe(rank))
    })

    it("marks the tiles' ids and counts the tiles before each word", () => {
        const {words, texels} = cellsOf(tiles, DETAIL)
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
        const doubled = new Float32Array([...tiles.subarray(0, 3), ...tiles.subarray(0, 3)])
        expect(() => cellsOf(doubled, DETAIL)).toThrow(/another tile/)
    })
})

describe("the shipped tiles", () => {
    it("each fall in their own cell", () => {
        const names = readdirSync(STATIC).filter((name) => /^landmask-[0-9a-f]{8}\.bin$/.test(name))
        expect(names).toHaveLength(1)
        const bytes = readFileSync(STATIC + names[0])
        const landmask = () => bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)

        const positions = rebuildTilePositions(landmask())
        const {isTile} = decodeLandmask(landmask())
        const ids = latticeIds(LATTICE_DETAIL)
        const faces = latticeFaces()

        let tile = 0
        for (let point = 0; point < latticeSize(LATTICE_DETAIL); point++) {
            if (!(isTile[point >> 3] & (1 << (point & 7)))) continue
            const found = cellOf(faces, LATTICE_COLS, ...pointOf(positions, tile))
            if (found !== ids[point]) expect(found, `tile ${tile + 1}`).toBe(ids[point])
            tile++
        }
        expect(tile).toBe(positions.length / 3)
        expect(() => cellsOf(positions)).not.toThrow()
    })
})

function popcount(bits: number): number {
    let count = 0
    for (let rest = bits >>> 0; rest; rest >>>= 1) count += rest & 1
    return count
}
