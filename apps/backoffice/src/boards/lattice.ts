// Where each tile of a board is, rebuilt from its runs (/map/README.md). The backoffice's own copy of the game page's
// lattice: the points of THREE.IcosahedronGeometry(1, cols - 1) in three's order, and only + - * /, Math.sqrt and
// Math.fround, so the positions come out bit for bit as the page's and hash to what the board's manifest says.

export type Lattice = {cols: number, warp: null, corners: null, order: "three"}

type Run = readonly [first: number, last: number]

const T = (1 + Math.sqrt(5)) / 2

const CORNERS: readonly number[] = [
    -1, T, 0, 1, T, 0, -1, -T, 0, 1, -T, 0,
    0, -1, T, 0, 1, T, 0, -1, -T, 0, 1, -T,
    T, 0, -1, T, 0, 1, -T, 0, -1, -T, 0, 1,
]

const FACES: readonly number[] = [
    0, 11, 5, 0, 5, 1, 0, 1, 7, 0, 7, 10, 0, 10, 11,
    1, 5, 9, 5, 11, 4, 11, 10, 2, 10, 7, 6, 7, 1, 8,
    3, 9, 4, 3, 4, 2, 3, 2, 6, 3, 6, 8, 3, 8, 9,
    4, 9, 5, 2, 4, 11, 6, 2, 10, 8, 6, 7, 9, 8, 1,
]

const CORNER_COUNT = 12
const EDGE_COUNT = 30
const FACE_COUNT = 20

const RUNS_MAGIC = "CPRN"
const RUNS_VERSION = 1
const RUNS_HEADER_BYTES = 12
const RUN_BYTES = 16

export function boardPositions(lattice: Lattice, tiles: number, runsFile: ArrayBuffer): Float32Array {
    if (lattice.warp !== null || lattice.corners !== null || lattice.order !== "three") {
        throw new Error("only a lattice with no warp, no rotation and three's order is read here")
    }
    const {cols} = lattice
    const isTile = new Uint8Array(10 * cols * cols + 2)
    for (const [first, last] of decodeRuns(runsFile)) {
        if (last >= isTile.length) throw new Error(`run ${first}-${last} is past the lattice's ${isTile.length} points`)
        isTile.fill(1, first, last + 1)
    }

    const positions = new Float32Array(tiles * 3)
    const faces = faceIds(cols)
    const seen = new Uint8Array(isTile.length)
    let tile = 0
    const push = (face: number, i: number, j: number) => {
        const id = canonicalId(cols, faces[face], i, j)
        if (seen[id]) return
        seen[id] = 1
        if (!isTile[id]) return
        if (tile === tiles) throw new Error(`the runs hold more than ${tiles} tiles`)
        writePosition(cols, face, i, j, positions, tile * 3)
        tile++
    }
    for (let face = 0; face < FACE_COUNT; face++) {
        for (let i = 0; i < cols; i++) {
            for (let j = 0; j < 2 * (cols - i) - 1; j++) {
                const k = Math.floor(j / 2)
                if (j % 2 === 0) {
                    push(face, i, k + 1)
                    push(face, i + 1, k)
                    push(face, i, k)
                } else {
                    push(face, i, k + 1)
                    push(face, i + 1, k + 1)
                    push(face, i + 1, k)
                }
            }
        }
    }
    if (tile !== tiles) throw new Error(`the runs hold ${tile} tiles, the board says ${tiles}`)
    return positions
}

export function positionsHash(positions: Float32Array): number {
    const words = new Uint32Array(positions.buffer, positions.byteOffset, positions.length)
    let hash = 0x811c9dc5
    for (let k = 0; k < words.length; k++) hash = Math.imul(hash ^ words[k], 0x01000193)
    return hash >>> 0
}

// "CPRN" | uint32 version | uint32 count | count × (uint64 first, uint64 last), canonical ids, ascending.
function decodeRuns(buffer: ArrayBuffer): Run[] {
    const view = new DataView(buffer)
    const magic = String.fromCharCode(...new Uint8Array(buffer, 0, Math.min(4, buffer.byteLength)))
    if (magic !== RUNS_MAGIC || buffer.byteLength < RUNS_HEADER_BYTES || view.getUint32(4, true) !== RUNS_VERSION) {
        throw new Error("not a runs file of version 1")
    }
    const count = view.getUint32(8, true)
    if (buffer.byteLength !== RUNS_HEADER_BYTES + count * RUN_BYTES) throw new Error("the runs file is truncated")

    const runs: Run[] = []
    for (let k = 0; k < count; k++) {
        const at = RUNS_HEADER_BYTES + k * RUN_BYTES
        if (view.getUint32(at + 4, true) !== 0 || view.getUint32(at + 12, true) !== 0) throw new Error("a run past 32 bits")
        runs.push([view.getUint32(at, true), view.getUint32(at + 8, true)])
    }
    return runs
}

type EdgeIds = {at: number, by: number}

type FaceIds = {corners: readonly [number, number, number], edges: readonly [EdgeIds, EdgeIds, EdgeIds], inside: number}

function faceIds(cols: number): FaceIds[] {
    const edgeOf = new Int8Array(CORNER_COUNT * CORNER_COUNT).fill(-1)
    let edges = 0
    for (let face = 0; face < FACE_COUNT; face++) {
        const a = FACES[face * 3], b = FACES[face * 3 + 1], c = FACES[face * 3 + 2]
        for (const [p, q] of [[a, b], [b, c], [a, c]]) {
            if (edgeOf[p * CORNER_COUNT + q] !== -1) continue
            edgeOf[p * CORNER_COUNT + q] = edgeOf[q * CORNER_COUNT + p] = edges++
        }
    }

    const edge = (from: number, to: number): EdgeIds => {
        const first = CORNER_COUNT + edgeOf[from * CORNER_COUNT + to] * (cols - 1) - 1
        return from < to ? {at: first, by: 1} : {at: first + cols, by: -1}
    }
    const insideFirst = CORNER_COUNT + EDGE_COUNT * (cols - 1)
    const insidePerFace = (cols - 2) * (cols - 1) / 2

    return Array.from({length: FACE_COUNT}, (_, face) => {
        const a = FACES[face * 3], b = FACES[face * 3 + 1], c = FACES[face * 3 + 2]
        return {corners: [a, b, c], edges: [edge(a, b), edge(a, c), edge(b, c)], inside: insideFirst + face * insidePerFace}
    })
}

function canonicalId(cols: number, {corners, edges, inside}: FaceIds, i: number, j: number): number {
    const rows = cols - i
    if (i === 0 && j === 0) return corners[0]
    if (i === 0 && j === cols) return corners[1]
    if (i === cols) return corners[2]
    if (i === 0) return edges[0].at + j * edges[0].by
    if (j === 0) return edges[1].at + i * edges[1].by
    if (j === rows) return edges[2].at + i * edges[2].by
    return inside + (i - 1) * (cols - 2) - (i - 1) * (i - 2) / 2 + j - 1
}

function writePosition(cols: number, face: number, i: number, j: number, out: Float32Array, at: number): void {
    const a = FACES[face * 3], b = FACES[face * 3 + 1], c = FACES[face * 3 + 2]
    const ax = CORNERS[a * 3], ay = CORNERS[a * 3 + 1], az = CORNERS[a * 3 + 2]
    const bx = CORNERS[b * 3], by = CORNERS[b * 3 + 1], bz = CORNERS[b * 3 + 2]
    const cx = CORNERS[c * 3], cy = CORNERS[c * 3 + 1], cz = CORNERS[c * 3 + 2]

    const down = i / cols
    const ajx = ax + (cx - ax) * down, ajy = ay + (cy - ay) * down, ajz = az + (cz - az) * down
    let x = ajx, y = ajy, z = ajz
    if (!(j === 0 && i === cols)) {
        const bjx = bx + (cx - bx) * down, bjy = by + (cy - by) * down, bjz = bz + (cz - bz) * down
        const across = j / (cols - i)
        x = ajx + (bjx - ajx) * across
        y = ajy + (bjy - ajy) * across
        z = ajz + (bjz - ajz) * across
    }

    const scale = 1 / (Math.sqrt(x * x + y * y + z * z) || 1)
    out[at] = Math.fround(x * scale)
    out[at + 1] = Math.fround(y * scale)
    out[at + 2] = Math.fround(z * scale)
}
