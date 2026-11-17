import {latticeSize, positionsHash, walkLattice, writePosition} from "./lattice.ts"
import {cellsOf, type TileCells} from "./latticeCells.ts"
import {chunksOf, type TileChunks} from "./tileChunks.ts"

// A board is a geometry: the lattice it is cut from, and which of its points are tiles (/map/README.md).
export type Board = {
    name: string
    body: string
    lattice: Lattice
    tiles: number
    // The median arc between two touching tiles, in radians.
    spacing: number
    positionsHash: number
    files: BoardFiles
}

// `earth-1` has no warp and no rotation, and numbers its tiles in three's order: the ledger holds those ids.
export type Lattice = {cols: number, warp: null, corners: null, order: "three"}

export type BoardFiles = {runs: string, borders: string, capitals: string, bridges: string}

// A board as the page carries it: the manifest, and where each of its files is served.
export type BoardAsset = Board & {urls: BoardFiles}

// Tiles side by side in one row of the lattice: the first and the last canonical id, both in.
export type Run = readonly [first: number, last: number]

// Where each tile is, its canonical id, and its place in three's walk of the lattice, all in tile order.
export type Tiles = {positions: Float32Array, ids: Uint32Array, points: Uint32Array}

// What the page draws from a board's runs: where each tile is, the cells the territory layer and the pick find a pixel's
// tile by, the chunks the page culls the tiles by, and the hash of the positions, which the board's manifest says.
export type RebuiltTiles = {positions: Float32Array, cells: TileCells, chunks: TileChunks, hash: number}

const RUNS_MAGIC = "CPRN"
const RUNS_VERSION = 1
const RUNS_HEADER_BYTES = 12
const RUN_BYTES = 16
const HIGH = 2 ** 32

export function tilesOf(lattice: Lattice, runs: readonly Run[], count: number): Tiles {
    if (lattice.warp !== null || lattice.corners !== null) throw new Error("a warped or rotated lattice is not built yet")
    if (lattice.order !== "three") throw new Error(`tile order ${String(lattice.order)} is not built yet`)

    const isTile = new Uint8Array(latticeSize(lattice.cols))
    for (const [first, last] of runs) {
        if (last >= isTile.length) throw new Error(`run ${first}-${last} is past the lattice's ${isTile.length} points`)
        isTile.fill(1, first, last + 1)
    }

    const positions = new Float32Array(count * 3)
    const ids = new Uint32Array(count)
    const points = new Uint32Array(count)
    let tile = 0
    walkLattice(lattice.cols, (point, face, i, j, id) => {
        if (!isTile[id]) return
        if (tile === count) throw new Error(`the runs hold more than ${count} tiles`)
        writePosition(lattice.cols, face, i, j, positions, tile * 3)
        ids[tile] = id
        points[tile] = point
        tile++
    })
    if (tile !== count) throw new Error(`the runs hold ${tile} tiles, the board says ${count}`)
    return {positions, ids, points}
}

export function rebuildTiles(lattice: Lattice, count: number, runs: ArrayBuffer): RebuiltTiles {
    const {positions, ids} = tilesOf(lattice, decodeRuns(runs), count)
    return {positions, cells: cellsOf(ids, lattice.cols), chunks: chunksOf(positions), hash: positionsHash(positions)}
}

// The runs of a set of canonical ids, ascending.
export function runsOf(ids: ArrayLike<number>): Run[] {
    const sorted = Float64Array.from(ids).sort()
    const runs: Run[] = []
    for (let k = 0; k < sorted.length; k++) {
        const id = sorted[k]
        const last = runs[runs.length - 1]
        if (last !== undefined && id <= last[1]) throw new Error(`canonical id ${id} is in twice`)
        if (last !== undefined && id === last[1] + 1) runs[runs.length - 1] = [last[0], id]
        else runs.push([id, id])
    }
    return runs
}

export function encodeRuns(runs: readonly Run[]): ArrayBuffer {
    checkRuns(runs)
    const buffer = new ArrayBuffer(RUNS_HEADER_BYTES + runs.length * RUN_BYTES)
    const view = new DataView(buffer)
    for (let i = 0; i < RUNS_MAGIC.length; i++) view.setUint8(i, RUNS_MAGIC.charCodeAt(i))
    view.setUint32(4, RUNS_VERSION, true)
    view.setUint32(8, runs.length, true)
    runs.forEach(([first, last], k) => {
        const at = RUNS_HEADER_BYTES + k * RUN_BYTES
        writeUint64(view, at, first)
        writeUint64(view, at + 8, last)
    })
    return buffer
}

export function decodeRuns(buffer: ArrayBuffer): Run[] {
    if (buffer.byteLength < RUNS_HEADER_BYTES) throw new Error(`runs file is truncated: ${buffer.byteLength} bytes`)
    const view = new DataView(buffer)
    let magic = ""
    for (let i = 0; i < RUNS_MAGIC.length; i++) magic += String.fromCharCode(view.getUint8(i))
    if (magic !== RUNS_MAGIC) throw new Error(`not a runs file: expected magic "${RUNS_MAGIC}", got "${magic}"`)

    const version = view.getUint32(4, true)
    if (version !== RUNS_VERSION) throw new Error(`unsupported runs format version ${version}, expected ${RUNS_VERSION}`)

    const count = view.getUint32(8, true)
    const expected = RUNS_HEADER_BYTES + count * RUN_BYTES
    if (buffer.byteLength !== expected) {
        throw new Error(`runs file size mismatch: ${count} runs need ${expected} bytes, got ${buffer.byteLength}`)
    }

    const runs: Run[] = new Array(count)
    for (let k = 0; k < count; k++) {
        const at = RUNS_HEADER_BYTES + k * RUN_BYTES
        runs[k] = [readUint64(view, at), readUint64(view, at + 8)]
    }
    checkRuns(runs)
    return runs
}

function checkRuns(runs: readonly Run[]): void {
    let after = -1
    for (const [first, last] of runs) {
        if (!Number.isSafeInteger(first) || !Number.isSafeInteger(last) || first < 0 || last < first) {
            throw new Error(`run ${first}-${last} is not a range of canonical ids`)
        }
        if (first <= after) throw new Error(`run ${first}-${last} is not after the run before it`)
        after = last
    }
}

function writeUint64(view: DataView, at: number, value: number): void {
    view.setUint32(at, value % HIGH, true)
    view.setUint32(at + 4, Math.floor(value / HIGH), true)
}

function readUint64(view: DataView, at: number): number {
    return view.getUint32(at, true) + view.getUint32(at + 4, true) * HIGH
}
