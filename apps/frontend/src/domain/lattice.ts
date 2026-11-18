// The points of a board's lattice: an icosahedron's 20 faces, each a triangular grid of `cols` steps per edge
// (/map/README.md). Two kinds:
//
// - three's: the points of THREE.IcosahedronGeometry(1, cols - 1), each once, in the order three generates them, with
//   three's own corners and arithmetic (`earth-1`, whose ids the ledger holds);
// - warped, rotated, or both, in canonical order (`earth-2`, `siege-1`): each point's barycentric weights warped so
//   every cell covers about as much ground, its face's corners turned once so the twelve five-neighbour points are at
//   sea (docs/boards.md, "The geometry").
//
// Positions must come out bit for bit on every engine: only + - * /, Math.sqrt and Math.fround, in the one order of
// operations written here, which every copy (the backoffice's, the backend's) follows.

const T = (1 + Math.sqrt(5)) / 2

// Three's icosahedron, unnormalised: three lerps the flat face first, so normalising here moves every tile.
export const CORNERS: readonly number[] = [
    -1, T, 0, 1, T, 0, -1, -T, 0, 1, -T, 0,
    0, -1, T, 0, 1, T, 0, -1, -T, 0, 1, -T,
    T, 0, -1, T, 0, 1, -T, 0, -1, -T, 0, 1,
]

export const FACES: readonly number[] = [
    0, 11, 5, 0, 5, 1, 0, 1, 7, 0, 7, 10, 0, 10, 11,
    1, 5, 9, 5, 11, 4, 11, 10, 2, 10, 7, 6, 7, 1, 8,
    3, 9, 4, 3, 4, 2, 3, 2, 6, 3, 6, 8, 3, 8, 9,
    4, 9, 5, 2, 4, 11, 6, 2, 10, 8, 6, 7, 9, 8, 1,
]

const CORNER_COUNT = 12
const EDGE_COUNT = 30
export const FACE_COUNT = 20

// The ±1 exchanges of a point's barycentric coordinates, as (i, j) steps on its face.
const MOVES: readonly (readonly [number, number])[] = [[0, 1], [0, -1], [1, 0], [-1, 0], [1, -1], [-1, 1]]

const MAX_DEGREE = 6

// The warp of a point's barycentric weights w = (wa, wb, wc): h_k = w_k·(1 + a·w_k + b·w_k²) + c·wa·wb·wc, then
// w'_k = h_k / (ha + hb + hc). Fitted per board so its cells cover about as much ground each (docs/boards.md).
export type Warp = {a: number, b: number, c: number}

// `three`: tile ids are ranks in three's walk of the lattice. `canonical`: ranks in canonical id order.
export type TileOrder = "three" | "canonical"

// A board's lattice, as its manifest says it: `warp` and `corners` null and the order `three` is three's own lattice.
// `corners` are the 36 float64s of three's corners turned once by the board's rotation, in the order of `CORNERS`.
export type Lattice = {cols: number, warp: Warp | null, corners: readonly number[] | null, order: TileOrder}

// Three's lattice at `cols`, which a bare number names wherever a lattice is asked for.
export function threeLattice(cols: number): Lattice {
    return {cols, warp: null, corners: null, order: "three"}
}

export function latticeOf(given: Lattice | number): Lattice {
    return typeof given === "number" ? threeLattice(given) : given
}

// Whether the positions are three's own arithmetic: no warp and no rotation.
export function isThrees(lattice: Lattice): boolean {
    return lattice.warp === null && lattice.corners === null
}

// The lattice's twelve corners, unnormalised: three's, or the board's rotated ones.
export function cornersOf(lattice: Lattice): readonly number[] {
    if (lattice.corners === null) return CORNERS
    if (lattice.corners.length !== CORNERS.length) {
        throw new Error(`a lattice's corners are ${CORNERS.length} numbers, not ${lattice.corners.length}`)
    }
    return lattice.corners
}

// `cols` steps along each edge of the icosahedron, so `cols + 1` points.
export function latticeSize(cols: number): number {
    return 10 * cols * cols + 2
}

// Every point's position, in the lattice's order.
export function latticePositions(given: Lattice | number): Float32Array {
    const lattice = latticeOf(given)
    const positions = new Float32Array(latticeSize(lattice.cols) * 3)
    const place = positionWriter(lattice)
    walkLattice(lattice, (point, face, i, j) => place(face, i, j, positions, point * 3))
    return positions
}

export function positionsHash(positions: Float32Array): number {
    const words = new Uint32Array(positions.buffer, positions.byteOffset, positions.length)
    let hash = 0x811c9dc5
    for (let k = 0; k < words.length; k++) hash = Math.imul(hash ^ words[k], 0x01000193)
    return hash >>> 0
}

// The canonical id of each point, in the lattice's order: in canonical order, each point's own place.
export function latticeIds(given: Lattice | number): Uint32Array {
    const lattice = latticeOf(given)
    const ids = new Uint32Array(latticeSize(lattice.cols))
    walkLattice(lattice, (point, _face, _i, _j, id) => {
        ids[point] = id
    })
    return ids
}

// Each point once, in the lattice's order: three's (the first time three's triangles reach it), or canonical.
export function walkLattice(given: Lattice | number, visit: (point: number, face: number, i: number, j: number, id: number) => void): void {
    const lattice = latticeOf(given)
    if (lattice.order === "canonical") {
        walkCanonical(lattice.cols, (face, i, j, id) => visit(id, face, i, j, id))
        return
    }
    if (lattice.order !== "three") throw new Error(`tile order ${String(lattice.order)} is not a lattice's`)
    const {cols} = lattice
    const faces = faceIds(cols)
    const seen = new Uint8Array(latticeSize(cols))
    let point = 0
    walkTriangles(cols, (face, i, j) => {
        const id = canonicalId(cols, faces[face], i, j)
        if (seen[id]) return
        seen[id] = 1
        visit(point++, face, i, j, id)
    })
}

// Each point once, in canonical id order: the 12 corners, then the 30 edges step by step from their lower corner, then
// each face's inside, row by row. A point on an edge or a corner comes out the same from every face it is on, to the
// bit, so the face it is reached from does not matter.
export function walkCanonical(cols: number, visit: (face: number, i: number, j: number, id: number) => void): void {
    const faces = faceIds(cols)
    // (i, j) of a face's corners a, b, c.
    const at: readonly (readonly [number, number])[] = [[0, 0], [0, cols], [cols, 0]]

    let id = 0
    for (let corner = 0; corner < CORNER_COUNT; corner++) {
        const face = Math.floor(FACES.indexOf(corner) / 3)
        const [i, j] = at[FACES.indexOf(corner) % 3]
        visit(face, i, j, id++)
    }

    const edges = edgeList()
    for (const [from, to] of edges) {
        let face = -1
        for (let f = 0; f < FACE_COUNT && face < 0; f++) {
            const corners = FACES.slice(f * 3, f * 3 + 3)
            if (corners.includes(from) && corners.includes(to)) face = f
        }
        const [fi, fj] = at[FACES.slice(face * 3, face * 3 + 3).indexOf(from)]
        const [ti, tj] = at[FACES.slice(face * 3, face * 3 + 3).indexOf(to)]
        const di = (ti - fi) / cols, dj = (tj - fj) / cols
        for (let step = 1; step < cols; step++) visit(face, fi + di * step, fj + dj * step, id++)
    }

    for (let face = 0; face < FACE_COUNT; face++) {
        if (id !== faces[face].inside) throw new Error(`face ${face}'s inside starts at ${faces[face].inside}, not ${id}`)
        for (let i = 1; i < cols - 1; i++) {
            for (let j = 1; j < cols - i; j++) visit(face, i, j, id++)
        }
    }
}

// Each triangle of the lattice, as points in the lattice's order, three per triangle: three's walk over each face.
export function latticeTriangles(given: Lattice | number): Uint32Array {
    const lattice = latticeOf(given)
    const {cols} = lattice
    const pointOf = pointsByCanonicalId(lattice)
    const faces = faceIds(cols)
    const corners = new Uint32Array(FACE_COUNT * cols * cols * 3)
    let at = 0
    walkTriangles(cols, (face, i, j) => {
        corners[at++] = pointOf[canonicalId(cols, faces[face], i, j)]
    })
    return corners
}

// Each point's neighbours, as points in the lattice's order: `to[at[p]]` to `to[at[p + 1]]`, ascending. Six, or five
// at the twelve corners; the ±1 exchanges of (i, j) on each face the point is on.
export function latticeNeighbours(given: Lattice | number): {at: Uint32Array, to: Uint32Array} {
    const lattice = latticeOf(given)
    const {cols} = lattice
    const points = latticeSize(cols)
    const pointOf = pointsByCanonicalId(lattice)
    const faces = faceIds(cols)
    const around = new Uint32Array(points * MAX_DEGREE)
    const degree = new Uint8Array(points)

    for (let face = 0; face < FACE_COUNT; face++) {
        for (let i = 0; i <= cols; i++) {
            for (let j = 0; j <= cols - i; j++) {
                const from = pointOf[canonicalId(cols, faces[face], i, j)]
                for (const [di, dj] of MOVES) {
                    const ni = i + di, nj = j + dj
                    if (ni < 0 || nj < 0 || ni + nj > cols) continue
                    const to = pointOf[canonicalId(cols, faces[face], ni, nj)]
                    const row = around.subarray(from * MAX_DEGREE, from * MAX_DEGREE + degree[from])
                    if (row.includes(to)) continue
                    if (degree[from] === MAX_DEGREE) throw new Error(`point ${from} has more than ${MAX_DEGREE} neighbours`)
                    around[from * MAX_DEGREE + degree[from]++] = to
                }
            }
        }
    }

    const at = new Uint32Array(points + 1)
    for (let point = 0; point < points; point++) at[point + 1] = at[point] + degree[point]
    const to = new Uint32Array(at[points])
    for (let point = 0; point < points; point++) {
        to.set(around.subarray(point * MAX_DEGREE, point * MAX_DEGREE + degree[point]).sort(), at[point])
    }
    return {at, to}
}

function pointsByCanonicalId(lattice: Lattice): Uint32Array {
    const pointOf = new Uint32Array(latticeSize(lattice.cols))
    walkLattice(lattice, (point, _face, _i, _j, id) => {
        pointOf[id] = point
    })
    return pointOf
}

// Three's own walk over each face's triangles (`PolyhedronGeometry`), one call per corner of each.
function walkTriangles(cols: number, push: (face: number, i: number, j: number) => void): void {
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

// The id of the point `step` along an edge is `at + step * by`.
export type EdgeIds = {at: number, by: number}

// One id per point of the sphere, whichever face it is reached from: a corner, a step along an edge, or a face's inside.
export type FaceIds = {
    corners: readonly [number, number, number]
    // a to b, a to c, b to c.
    edges: readonly [EdgeIds, EdgeIds, EdgeIds]
    inside: number
}

export function faceIds(cols: number): FaceIds[] {
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
        return {
            corners: [a, b, c],
            edges: [edge(a, b), edge(a, c), edge(b, c)],
            inside: insideFirst + face * insidePerFace,
        }
    })
}

export function canonicalId(cols: number, {corners, edges, inside}: FaceIds, i: number, j: number): number {
    const rows = cols - i
    if (i === 0 && j === 0) return corners[0]
    if (i === 0 && j === cols) return corners[1]
    if (i === cols) return corners[2]
    if (i === 0) return edges[0].at + j * edges[0].by
    if (j === 0) return edges[1].at + i * edges[1].by
    if (j === rows) return edges[2].at + i * edges[2].by
    return inside + (i - 1) * (cols - 2) - (i - 1) * (i - 2) / 2 + j - 1
}

export type PositionWriter = (face: number, i: number, j: number, out: Float32Array, at: number) => void

// Writes the position of (face, i, j) as three float32s: three's lerps for three's lattice, the warped barycentric
// weights over the rotated corners otherwise.
export function positionWriter(given: Lattice | number): PositionWriter {
    const lattice = latticeOf(given)
    const {cols} = lattice
    if (isThrees(lattice)) return (face, i, j, out, at) => writeThreesPosition(cols, face, i, j, out, at)
    const corners = cornersOf(lattice)
    const {warp} = lattice
    const weights = new Float64Array(3)
    return (face, i, j, out, at) => {
        warpedWeights(warp, (cols - i - j) / cols, j / cols, i / cols, weights)
        writeOnCorners(corners, face, weights[0], weights[1], weights[2], out, at)
    }
}

// The warp of barycentric weights, into `out`: h_k = w_k·((1 + a·w_k) + b·(w_k·w_k)) + ((c·wa)·wb)·wc, then each over
// (ha + hb) + hc. No warp leaves them as they are.
export function warpedWeights(warp: Warp | null, wa: number, wb: number, wc: number, out: Float64Array): void {
    if (warp === null) {
        out[0] = wa
        out[1] = wb
        out[2] = wc
        return
    }
    const {a, b, c} = warp
    const shared = c * wa * wb * wc
    const ha = wa * (1 + a * wa + b * (wa * wa)) + shared
    const hb = wb * (1 + a * wb + b * (wb * wb)) + shared
    const hc = wc * (1 + a * wc + b * (wc * wc)) + shared
    const sum = ha + hb + hc
    out[0] = ha / sum
    out[1] = hb / sum
    out[2] = hc / sum
}

// (wa·A + wb·B) + wc·C over the face's corners, then normalised as three does: × 1 / length.
function writeOnCorners(corners: readonly number[], face: number, wa: number, wb: number, wc: number, out: Float32Array, at: number): void {
    const a = FACES[face * 3] * 3, b = FACES[face * 3 + 1] * 3, c = FACES[face * 3 + 2] * 3
    const x = wa * corners[a] + wb * corners[b] + wc * corners[c]
    const y = wa * corners[a + 1] + wb * corners[b + 1] + wc * corners[c + 1]
    const z = wa * corners[a + 2] + wb * corners[b + 2] + wc * corners[c + 2]
    const scale = 1 / Math.sqrt(x * x + y * y + z * z)
    out[at] = Math.fround(x * scale)
    out[at + 1] = Math.fround(y * scale)
    out[at + 2] = Math.fround(z * scale)
}

// Three's own: each lerp as a + (c - a) * t, then × 1 / length.
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

// The weights w whose warp is (·, warpedB, warpedC), into `out` as (wb, wc): two Newton steps from w = w'. A cell is
// 1/cols of a weight wide; two steps land within about 1e-7 of it (latticeCells.test.ts).
export function unwarp(warp: Warp, warpedB: number, warpedC: number, out: Float64Array, steps = 2): void {
    const {a, b, c} = warp
    let u = warpedB, v = warpedC
    for (let step = 0; step < steps; step++) {
        const wa = 1 - u - v
        const shared = c * wa * u * v
        const ha = wa * (1 + a * wa + b * wa * wa) + shared
        const hb = u * (1 + a * u + b * u * u) + shared
        const hc = v * (1 + a * v + b * v * v) + shared
        const sum = ha + hb + hc
        // ∂h_k/∂w_k, and the shared term's slope along u (wb) and v (wc), wa = 1 − u − v.
        const da = 1 + 2 * a * wa + 3 * b * wa * wa
        const db = 1 + 2 * a * u + 3 * b * u * u
        const dc = 1 + 2 * a * v + 3 * b * v * v
        const su = c * v * (wa - u), sv = c * u * (wa - v)
        const hbU = db + su, hbV = sv
        const hcU = su, hcV = dc + sv
        const sumU = -da + db + 3 * su, sumV = -da + dc + 3 * sv
        // The warped wb' and wc', and their slopes.
        const fu = hb / sum - warpedB, fv = hc / sum - warpedC
        const j00 = (hbU * sum - hb * sumU) / (sum * sum), j01 = (hbV * sum - hb * sumV) / (sum * sum)
        const j10 = (hcU * sum - hc * sumU) / (sum * sum), j11 = (hcV * sum - hc * sumV) / (sum * sum)
        const det = j00 * j11 - j01 * j10
        u -= (j11 * fu - j01 * fv) / det
        v -= (j00 * fv - j10 * fu) / det
    }
    out[0] = u
    out[1] = v
}
