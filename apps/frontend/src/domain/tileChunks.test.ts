import {readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {BOARD} from "../app/viewer/boardAsset.ts"
import {decodeRuns, tilesOf} from "./board.ts"
import {
    CAP_FLOATS,
    CHUNK_REACH,
    chunkCount,
    chunkOf,
    chunksOf,
    CHUNKS_PER_BLOCK,
    type ChunkView,
    drawnRanges,
    showingChunks,
} from "./tileChunks.ts"

const STATIC = fileURLToPath(new URL("../../static/", import.meta.url))

function shippedPositions(): Float32Array {
    const bytes = readFileSync(`${STATIC}boards/${BOARD.name}/${BOARD.files.runs}`)
    return tilesOf(BOARD.lattice, decodeRuns(bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)), BOARD.tiles).positions
}

const positions = shippedPositions()
const tiles = positions.length / 3
const chunks = chunksOf(positions)
const count = chunkCount(chunks)

function angleTo(x: number, y: number, z: number, caps: Float32Array, at: number): number {
    const centre = [caps[at], caps[at + 1], caps[at + 2]]
    return Math.atan2(Math.hypot(...cross([x, y, z], centre)), x * centre[0] + y * centre[1] + z * centre[2])
}

describe("the chunks of the shipped tiles", () => {
    it("hold every tile in exactly one chunk, in order", () => {
        expect(chunks.starts[0]).toBe(0)
        expect(chunks.starts[count]).toBe(tiles)
        for (let chunk = 0; chunk < count; chunk++) expect(chunks.starts[chunk + 1]).toBeGreaterThan(chunks.starts[chunk])
    })

    it("give each chunk a cap holding all its tiles", () => {
        for (let chunk = 0; chunk < count; chunk++) {
            const at = chunk * CAP_FLOATS
            for (let tile = chunks.starts[chunk]; tile < chunks.starts[chunk + 1]; tile++) {
                const angle = angleTo(positions[tile * 3], positions[tile * 3 + 1], positions[tile * 3 + 2], chunks.caps, at)
                if (angle > chunks.caps[at + 3]) expect(angle, `tile ${tile + 1} in chunk ${chunk}`).toBeLessThanOrEqual(chunks.caps[at + 3])
            }
        }
    })

    it("keep each cap small: no tile is further than the reach from its chunk's first", () => {
        for (let chunk = 0; chunk < count; chunk++) expect(chunks.caps[chunk * CAP_FLOATS + 3]).toBeLessThanOrEqual(2 * CHUNK_REACH)
    })

    it("give each block a cap holding its chunks' caps", () => {
        for (let chunk = 0; chunk < count; chunk++) {
            const at = chunk * CAP_FLOATS
            const block = Math.floor(chunk / CHUNKS_PER_BLOCK) * CAP_FLOATS
            const reach = angleTo(chunks.caps[at], chunks.caps[at + 1], chunks.caps[at + 2], chunks.blocks, block) + chunks.caps[at + 3]
            expect(reach, `chunk ${chunk}`).toBeLessThanOrEqual(chunks.blocks[block + 3])
        }
    })

    it("find the chunk of a tile", () => {
        for (let tile = 0; tile < tiles; tile += 211) {
            const chunk = chunkOf(chunks, tile)
            expect(chunks.starts[chunk]).toBeLessThanOrEqual(tile)
            expect(chunks.starts[chunk + 1]).toBeGreaterThan(tile)
        }
        expect(chunkOf(chunks, 0)).toBe(0)
        expect(chunkOf(chunks, tiles - 1)).toBe(count - 1)
    })
})

type Camera = {view: ChunkView, ground: (x: number, y: number, z: number) => {x: number, y: number, facing: number}}

function cameraOf(direction: [number, number, number], zoom: number, aspect: number, limb: number, reach: number): Camera {
    const length = Math.hypot(...direction)
    const back = direction.map((v) => v / length) as [number, number, number]
    const side: [number, number, number] = Math.abs(back[1]) < 0.9 ? [0, 1, 0] : [1, 0, 0]
    const right = normalized(cross(side, back))
    const up = cross(back, right)
    const sx = zoom / aspect, sy = zoom
    const toClip = [
        right[0] * sx, up[0] * sy, 0, 0,
        right[1] * sx, up[1] * sy, 0, 0,
        right[2] * sx, up[2] * sy, 0, 0,
        0, 0, 0, 1,
    ]
    return {
        view: {toClip, back, limb, reach},
        ground: (x, y, z) => ({
            x: (x * right[0] + y * right[1] + z * right[2]) * sx,
            y: (x * up[0] + y * up[1] + z * up[2]) * sy,
            facing: (x * back[0] + y * back[1] + z * back[2]) / Math.hypot(x, y, z),
        }),
    }
}

describe("the chunks in sight", () => {
    const directions: [number, number, number][] = [[1, 0.3, 0.2], [-0.4, 0.8, -0.1], [0.1, -0.9, 0.5], [-0.7, -0.2, -0.7], [0, 0, 1]]
    const zooms = [0.5, 1, 3, 10, 40]

    it("hold every tile the display shader would draw", () => {
        const shown = new Uint32Array(count)
        for (const direction of directions) {
            for (const zoom of zooms) {
                const limb = 0.06, reach = 0.02 / zoom
                const {view, ground} = cameraOf(direction, zoom, 1.6, limb, reach)
                const kept = new Uint8Array(count)
                const written = showingChunks(chunks, view, shown)
                for (let i = 0; i < written; i++) kept[shown[i]] = 1

                const sx = zoom / 1.6, sy = zoom
                let missed = 0
                for (let tile = 0; tile < tiles; tile++) {
                    const {x, y, facing} = ground(positions[tile * 3], positions[tile * 3 + 1], positions[tile * 3 + 2])
                    const drawn = facing >= -limb && Math.abs(x) <= 1 + sx * reach && Math.abs(y) <= 1 + sy * reach
                    if (drawn && !kept[chunkOf(chunks, tile)]) missed++
                }
                expect(missed, `zoom ${zoom} from ${direction}`).toBe(0)
            }
        }
    })

    it("are far fewer than the map when the camera is close", () => {
        const shown = new Uint32Array(count)
        const {view} = cameraOf([1, 0.3, 0.2], 40, 1.6, 0.06, 0.001)
        const written = showingChunks(chunks, view, shown)
        const drawn = drawnRanges(chunks, shown, written, 16).reduce((sum, {count}) => sum + count, 0)
        expect(drawn).toBeLessThan(tiles / 50)
    })

    it("skip the chunks a filter leaves out", () => {
        const shown = new Uint32Array(count)
        const {view} = cameraOf([0, 0, 1], 1, 1.6, 0.06, 0.01)
        const written = showingChunks(chunks, view, shown, (chunk) => chunk % 7 === 0)
        expect(written).toBeGreaterThan(0)
        for (let i = 0; i < written; i++) expect(shown[i] % 7).toBe(0)
    })
})

describe("drawnRanges", () => {
    const shownOf = (...indices: number[]) => Uint32Array.from(indices)

    it("covers exactly the tiles of the chunks in sight, a run of chunks as one range", () => {
        const ranges = drawnRanges(chunks, shownOf(2, 3, 4, 9), 4, 16, 0)
        expect(ranges).toEqual([
            {start: chunks.starts[2], count: chunks.starts[5] - chunks.starts[2]},
            {start: chunks.starts[9], count: chunks.starts[10] - chunks.starts[9]},
        ])
    })

    it("joins chunks closer than the least gap", () => {
        const ranges = drawnRanges(chunks, shownOf(2, 9), 2, 16, tiles)
        expect(ranges).toEqual([{start: chunks.starts[2], count: chunks.starts[10] - chunks.starts[2]}])
    })

    it("draws nothing when nothing is in sight", () => {
        expect(drawnRanges(chunks, shownOf(), 0, 16)).toEqual([])
    })
})

function cross(a: readonly number[], b: readonly number[]): [number, number, number] {
    return [a[1] * b[2] - a[2] * b[1], a[2] * b[0] - a[0] * b[2], a[0] * b[1] - a[1] * b[0]]
}

function normalized(a: [number, number, number]): [number, number, number] {
    const length = Math.hypot(...a)
    return [a[0] / length, a[1] / length, a[2] / length]
}
