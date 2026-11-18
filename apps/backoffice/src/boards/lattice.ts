// Where each tile of a board is, rebuilt from its runs (/map/README.md). The backoffice's own copy of the game page's
// lattice (its domain/lattice.ts): three's lattice, the points of THREE.IcosahedronGeometry(1, cols - 1) in three's
// order (earth-1), or a warped and rotated one in canonical order (earth-2, siege-1). Only + - * /, Math.sqrt and
// Math.fround, in the page's order of operations, so the positions come out bit for bit as the page's and hash to what
// the board's manifest says.

// The warp of a point's barycentric weights w: h_k = w_k·(1 + a·w_k + b·w_k²) + c·wa·wb·wc, then each over their sum.
export type Warp = {a: number, b: number, c: number}

// A board's lattice: `warp` and `corners` null and the order `three` is three's; `corners` are three's 12 corners turned
// by the board's rotation, 36 numbers.
export type Lattice = {cols: number, warp: Warp | null, corners: readonly number[] | null, order: "three" | "canonical"}

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
    if (lattice.order !== "three" && lattice.order !== "canonical") throw new Error(`tile order ${String(lattice.order)} is no lattice's`)
    if (lattice.corners !== null && lattice.corners.length !== CORNERS.length) {
        throw new Error(`a lattice's corners are ${CORNERS.length} numbers, not ${lattice.corners.length}`)
    }
    const {cols} = lattice
    const isTile = new Uint8Array(10 * cols * cols + 2)
    for (const [first, last] of decodeRuns(runsFile)) {
        if (last >= isTile.length) throw new Error(`run ${first}-${last} is past the lattice's ${isTile.length} points`)
        isTile.fill(1, first, last + 1)
    }

    const positions = new Float32Array(tiles * 3)
    const place = positionWriter(lattice)
    let tile = 0
    // A tile's id is its rank among the tiles in the board's order, plus one.
    walk(lattice, (face, i, j, id) => {
        if (!isTile[id]) return
        if (tile === tiles) throw new Error(`the runs hold more than ${tiles} tiles`)
        place(face, i, j, positions, tile * 3)
        tile++
    })
    if (tile !== tiles) throw new Error(`the runs hold ${tile} tiles, the board says ${tiles}`)
    return positions
}

// Each point once, in the board's order: three's walk (the first time three's triangles reach it), or canonical.
function walk({cols, order}: Lattice, visit: (face: number, i: number, j: number, id: number) => void): void {
    const faces = faceIds(cols)
    if (order === "canonical") {
        walkCanonical(cols, faces, visit)
        return
    }
    const seen = new Uint8Array(10 * cols * cols + 2)
    const push = (face: number, i: number, j: number) => {
        const id = canonicalId(cols, faces[face], i, j)
        if (seen[id]) return
        seen[id] = 1
        visit(face, i, j, id)
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
}

// The 12 corners, then the 30 edges step by step from their lower corner, then each face's inside, row by row. A point
// on an edge or a corner comes out the same from every face it is on, so the face it is reached from does not matter.
function walkCanonical(cols: number, faces: FaceIds[], visit: (face: number, i: number, j: number, id: number) => void): void {
    const at: readonly (readonly [number, number])[] = [[0, 0], [0, cols], [cols, 0]]
    const cornersOf = (face: number) => FACES.slice(face * 3, face * 3 + 3)
    let id = 0
    for (let corner = 0; corner < CORNER_COUNT; corner++) {
        const face = Math.floor(FACES.indexOf(corner) / 3)
        const [i, j] = at[FACES.indexOf(corner) % 3]
        visit(face, i, j, id++)
    }
    for (const [from, to] of edgeList()) {
        let face = 0
        while (!cornersOf(face).includes(from) || !cornersOf(face).includes(to)) face++
        const [fi, fj] = at[cornersOf(face).indexOf(from)]
        const [ti, tj] = at[cornersOf(face).indexOf(to)]
        for (let step = 1; step < cols; step++) visit(face, fi + (ti - fi) / cols * step, fj + (tj - fj) / cols * step, id++)
    }
    for (let face = 0; face < FACE_COUNT; face++) {
        if (id !== faces[face].inside) throw new Error(`face ${face}'s inside starts at ${faces[face].inside}, not ${id}`)
        for (let i = 1; i < cols - 1; i++) {
            for (let j = 1; j < cols - i; j++) visit(face, i, j, id++)
        }
    }
}

// The 30 edges, as their two corners, the lower first, in the order their canonical ids run.
function edgeList(): [number, number][] {
    const seen = new Set<number>()
    const edges: [number, number][] = []
    for (let face = 0; face < FACE_COUNT; face++) {
        const a = FACES[face * 3], b = FACES[face * 3 + 1], c = FACES[face * 3 + 2]
        for (const [p, q] of [[a, b], [b, c], [a, c]]) {
            const key = Math.min(p, q) * CORNER_COUNT + Math.max(p, q)
            if (seen.has(key)) continue
            seen.add(key)
            edges.push([Math.min(p, q), Math.max(p, q)])
        }
    }
    return edges
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
    edgeList().forEach(([p, q], edge) => {
        edgeOf[p * CORNER_COUNT + q] = edgeOf[q * CORNER_COUNT + p] = edge
    })

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

type PositionWriter = (face: number, i: number, j: number, out: Float32Array, at: number) => void

// Three's lerps for three's lattice; the warped barycentric weights over the rotated corners otherwise.
function positionWriter({cols, warp, corners}: Lattice): PositionWriter {
    if (warp === null && corners === null) return (face, i, j, out, at) => writeThreesPosition(cols, face, i, j, out, at)
    const on = corners ?? CORNERS
    return (face, i, j, out, at) => {
        let wa = (cols - i - j) / cols, wb = j / cols, wc = i / cols
        if (warp !== null) {
            // h_k = w_k·((1 + a·w_k) + b·(w_k·w_k)) + ((c·wa)·wb)·wc, then each over (ha + hb) + hc.
            const shared = warp.c * wa * wb * wc
            const ha = wa * (1 + warp.a * wa + warp.b * (wa * wa)) + shared
            const hb = wb * (1 + warp.a * wb + warp.b * (wb * wb)) + shared
            const hc = wc * (1 + warp.a * wc + warp.b * (wc * wc)) + shared
            const sum = ha + hb + hc
            wa = ha / sum
            wb = hb / sum
            wc = hc / sum
        }
        const a = FACES[face * 3] * 3, b = FACES[face * 3 + 1] * 3, c = FACES[face * 3 + 2] * 3
        const x = wa * on[a] + wb * on[b] + wc * on[c]
        const y = wa * on[a + 1] + wb * on[b + 1] + wc * on[c + 1]
        const z = wa * on[a + 2] + wb * on[b + 2] + wc * on[c + 2]
        const scale = 1 / Math.sqrt(x * x + y * y + z * z)
        out[at] = Math.fround(x * scale)
        out[at + 1] = Math.fround(y * scale)
        out[at + 2] = Math.fround(z * scale)
    }
}

function writeThreesPosition(cols: number, face: number, i: number, j: number, out: Float32Array, at: number): void {
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
