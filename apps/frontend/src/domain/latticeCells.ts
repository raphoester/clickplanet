import {
    canonicalId,
    cornersOf,
    FACE_COUNT,
    FACES,
    faceIds,
    type FaceIds,
    type Lattice,
    latticeOf,
    latticeSize,
    unwarp,
} from "./lattice.ts"

// Must match the territory layer's fragment shader, which finds a pixel's cell the same way: the face the direction
// leaves the icosahedron through, the gnomonic inverse onto that face's plane, which gives the warped weights w', then
// two Newton steps back to the lattice's own weights w on a warped lattice, and the nearest point of the face's grid.

export type Vec3 = readonly [number, number, number]

export type LatticeFace = {
    normal: Vec3
    height: number
    origin: Vec3
    toB: Vec3
    toC: Vec3
    // A point of the face's plane is origin + toB * along + toC * down.
    alongOf: Vec3
    downOf: Vec3
    ids: FaceIds
}

export function latticeFaces(given: Lattice | number): LatticeFace[] {
    const lattice = latticeOf(given)
    const corners = cornersOf(lattice)
    const ids = faceIds(lattice.cols)
    return Array.from({length: FACE_COUNT}, (_, face) => {
        const [a, b, c] = [0, 1, 2].map((k) => cornerOf(corners, FACES[face * 3 + k]))
        const toB = minus(b, a)
        const toC = minus(c, a)
        const normal = normalized(plus(plus(a, b), c))

        const bb = dot(toB, toB), bc = dot(toB, toC), cc = dot(toC, toC)
        const det = bb * cc - bc * bc
        return {
            normal,
            height: dot(a, normal),
            origin: a,
            toB,
            toC,
            alongOf: scaled(minus(scaled(toB, cc), scaled(toC, bc)), 1 / det),
            downOf: scaled(minus(scaled(toC, bb), scaled(toB, bc)), 1 / det),
            ids: ids[face],
        }
    })
}

// Each face's normal, height, origin, alongOf and downOf, packed: the pick and `cellsOf` find millions of cells.
const PACKED = 13
const packedFaces = new WeakMap<readonly LatticeFace[], Float64Array>()

function packed(faces: readonly LatticeFace[]): Float64Array {
    let numbers = packedFaces.get(faces)
    if (numbers === undefined) {
        numbers = new Float64Array(faces.length * PACKED)
        faces.forEach(({normal, height, origin, alongOf, downOf}, face) => {
            numbers!.set([...normal, height, ...origin, ...alongOf, ...downOf], face * PACKED)
        })
        packedFaces.set(faces, numbers)
    }
    return numbers
}

const unwarped = new Float64Array(2)

// The lattice point nearest a direction, as its canonical id, on the face's grid of the lattice's own weights: a cell
// is a few thousandths of a radian wide. The face is the one the direction leaves the icosahedron through: every face
// is as far from the centre.
export function cellOf(faces: readonly LatticeFace[], given: Lattice | number, x: number, y: number, z: number): number {
    const cols = typeof given === "number" ? given : given.cols
    const warp = typeof given === "number" ? null : given.warp
    const numbers = packed(faces)
    let face = 0
    let best = -Infinity
    for (let f = 0, at = 0; f < faces.length; f++, at += PACKED) {
        const facing = x * numbers[at] + y * numbers[at + 1] + z * numbers[at + 2]
        if (facing > best) {
            best = facing
            face = f
        }
    }
    const at = face * PACKED
    const reach = numbers[at + 3] / best
    const qx = x * reach - numbers[at + 4], qy = y * reach - numbers[at + 5], qz = z * reach - numbers[at + 6]
    let along = qx * numbers[at + 7] + qy * numbers[at + 8] + qz * numbers[at + 9]
    let down = qx * numbers[at + 10] + qy * numbers[at + 11] + qz * numbers[at + 12]
    if (warp !== null) {
        unwarp(warp, along, down, unwarped)
        along = unwarped[0]
        down = unwarped[1]
    }
    const point = nearestPoint(along * cols, down * cols)
    const i = Math.floor(point / GRID_SPAN) - GRID_OFFSET, j = point % GRID_SPAN - GRID_OFFSET

    const row = Math.min(Math.max(i, 0), cols)
    return canonicalId(cols, faces[face].ids, row, Math.min(Math.max(j, 0), cols - row))
}

// nearestOnGrid's (x, y) as one number, (y + GRID_OFFSET) * GRID_SPAN + x + GRID_OFFSET: no array per cell.
const GRID_SPAN = 1 << 20
const GRID_OFFSET = 4

function nearestPoint(x: number, y: number): number {
    const fx = Math.floor(x), fy = Math.floor(y)
    const rx = x - fx, ry = y - fy
    let px = fx, py = fy
    if (rx + ry < 1) {
        const rest = 1 - rx - ry
        if (!(rest >= rx && rest >= ry)) {
            if (rx >= ry) px++
            else py++
        }
    } else {
        const far = rx + ry - 1
        const right = 1 - ry, up = 1 - rx
        if (far >= right && far >= up) {
            px++
            py++
        } else if (right >= up) {
            px++
        } else {
            py++
        }
    }
    return (py + GRID_OFFSET) * GRID_SPAN + px + GRID_OFFSET
}

// The nearest point of a grid of equilateral triangles cut along their short diagonal: the corner with the largest weight.
export function nearestOnGrid(x: number, y: number): [number, number] {
    const point = nearestPoint(x, y)
    return [point % GRID_SPAN - GRID_OFFSET, Math.floor(point / GRID_SPAN) - GRID_OFFSET]
}

// One pair per 32 canonical ids of the lattice: which are tiles, and how many tiles come before them. A tile's texel is
// its place among the tiles in canonical order: on a board in canonical order, its own index.
export type TileCells = {
    lattice: Lattice
    words: Uint32Array
    texels: Uint32Array
    // The inverse of texels: the 0-based tile at each texel.
    tiles: Uint32Array
}

export const IDS_PER_WORD = 32

// `ids` holds each tile's canonical id, in tile order.
export function cellsOf(ids: Uint32Array, given: Lattice | number): TileCells {
    const lattice = latticeOf(given)
    const points = latticeSize(lattice.cols)
    const isTile = new Uint8Array(points)
    ids.forEach((id, tile) => {
        if (id >= points) throw new Error(`tile ${tile + 1} is past the lattice's ${points} points`)
        if (isTile[id]) throw new Error(`tile ${tile + 1} is in the cell of another tile`)
        isTile[id] = 1
    })

    const wordCount = Math.ceil(points / IDS_PER_WORD)
    const words = new Uint32Array(wordCount * 2)
    const rankOf = new Uint32Array(points)
    let rank = 0
    for (let word = 0; word < wordCount; word++) {
        words[word * 2 + 1] = rank
        let bits = 0
        for (let bit = 0; bit < IDS_PER_WORD; bit++) {
            const id = word * IDS_PER_WORD + bit
            if (id >= points || !isTile[id]) continue
            bits |= 1 << bit
            rankOf[id] = rank++
        }
        words[word * 2] = bits >>> 0
    }

    const texels = ids.map((id) => rankOf[id])
    const byTexel = new Uint32Array(ids.length)
    texels.forEach((texel, tile) => {
        byTexel[texel] = tile
    })
    return {lattice, words, texels, tiles: byTexel}
}

// Every tile's position must find its own cell, as the territory layer and the pick find a pixel's: a lattice whose
// inverse misses one is refused. `ids` holds each tile's canonical id, and `positions` its position, in tile order.
export function checkCells(ids: Uint32Array, given: Lattice | number, positions: Float32Array): void {
    const lattice = latticeOf(given)
    if (positions.length !== ids.length * 3) throw new Error(`${ids.length} tiles but ${positions.length / 3} positions`)
    const faces = latticeFaces(lattice)
    ids.forEach((id, tile) => {
        const found = cellOf(faces, lattice, positions[tile * 3], positions[tile * 3 + 1], positions[tile * 3 + 2])
        if (found !== id) throw new Error(`tile ${tile + 1}, point ${id}, is found in the cell of point ${found}`)
    })
}

// The tile whose cell holds a direction, by its 1-based id, or undefined over the sea.
export function tileAt(cells: TileCells, faces: readonly LatticeFace[], x: number, y: number, z: number): number | undefined {
    const id = cellOf(faces, cells.lattice, x, y, z)
    const bits = cells.words[(id >>> 5) * 2]
    const bit = id & 31
    if (!((bits >>> bit) & 1)) return undefined
    const texel = cells.words[(id >>> 5) * 2 + 1] + popcount(bits & ((1 << bit) - 1))
    return cells.tiles[texel] + 1
}

function popcount(bits: number): number {
    bits = bits - ((bits >>> 1) & 0x55555555)
    bits = (bits & 0x33333333) + ((bits >>> 2) & 0x33333333)
    return (Math.imul((bits + (bits >>> 4)) & 0x0F0F0F0F, 0x01010101) >>> 24)
}

function cornerOf(corners: readonly number[], corner: number): Vec3 {
    return [corners[corner * 3], corners[corner * 3 + 1], corners[corner * 3 + 2]]
}

function plus(a: Vec3, b: Vec3): Vec3 {
    return [a[0] + b[0], a[1] + b[1], a[2] + b[2]]
}

function minus(a: Vec3, b: Vec3): Vec3 {
    return [a[0] - b[0], a[1] - b[1], a[2] - b[2]]
}

function scaled(a: Vec3, by: number): Vec3 {
    return [a[0] * by, a[1] * by, a[2] * by]
}

function dot(a: Vec3, b: Vec3): number {
    return a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

function normalized(a: Vec3): Vec3 {
    return scaled(a, 1 / Math.sqrt(dot(a, a)))
}
