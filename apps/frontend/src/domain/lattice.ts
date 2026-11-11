// The vertices of THREE.IcosahedronGeometry(1, detail), each once, in the order three generates them.
// Positions must come out bit for bit as three's: only + - * /, Math.sqrt and three's own lerp and normalize forms.

export const LATTICE_DETAIL = 300

const T = (1 + Math.sqrt(5)) / 2

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

export function latticeSize(detail: number): number {
    const cols = detail + 1
    return 10 * cols * cols + 2
}

export function latticePositions(detail: number): Float32Array {
    const positions = new Float32Array(latticeSize(detail) * 3)
    walkLattice(detail, (point, face, i, j) => writePosition(detail + 1, face, i, j, positions, point * 3))
    return positions
}

// `isTile` holds one bit per lattice point, lowest bit first; the result holds the kept points' positions in order.
export function tilePositions(detail: number, isTile: Uint8Array, tiles: number): Float32Array {
    const positions = new Float32Array(tiles * 3)
    let tile = 0
    walkLattice(detail, (point, face, i, j) => {
        if (!(isTile[point >> 3] & (1 << (point & 7)))) return
        if (tile === tiles) throw new Error(`the mask keeps more than ${tiles} tiles`)
        writePosition(detail + 1, face, i, j, positions, tile * 3)
        tile++
    })
    if (tile !== tiles) throw new Error(`the mask keeps ${tile} tiles, expected ${tiles}`)
    return positions
}

export function positionsHash(positions: Float32Array): number {
    const words = new Uint32Array(positions.buffer, positions.byteOffset, positions.length)
    let hash = 0x811c9dc5
    for (let k = 0; k < words.length; k++) hash = Math.imul(hash ^ words[k], 0x01000193)
    return hash >>> 0
}

// The canonical id of each point, in the order three generates them.
export function latticeIds(detail: number): Uint32Array {
    const ids = new Uint32Array(latticeSize(detail))
    walkLattice(detail, (point, _face, _i, _j, id) => {
        ids[point] = id
    })
    return ids
}

function walkLattice(detail: number, visit: (point: number, face: number, i: number, j: number, id: number) => void): void {
    const cols = detail + 1
    const faces = faceIds(detail)
    const seen = new Uint8Array(latticeSize(detail))
    let point = 0
    const push = (face: number, i: number, j: number) => {
        const id = canonicalId(cols, faces[face], i, j)
        if (seen[id]) return
        seen[id] = 1
        visit(point++, face, i, j, id)
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

// The id of the point `step` along an edge is `at + step * by`.
export type EdgeIds = {at: number, by: number}

// One id per point of the sphere, whichever face it is reached from: a corner, a step along an edge, or a face's inside.
export type FaceIds = {
    corners: readonly [number, number, number]
    // a to b, a to c, b to c.
    edges: readonly [EdgeIds, EdgeIds, EdgeIds]
    inside: number
}

export function faceIds(detail: number): FaceIds[] {
    const cols = detail + 1
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
