import {canonicalId, CORNERS, FACE_COUNT, FACES, faceIds, type FaceIds, latticeSize} from "./lattice.ts"

// Must match the territory layer's fragment shader, which finds a pixel's cell the same way.

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

export function latticeFaces(cols: number): LatticeFace[] {
    const ids = faceIds(cols)
    return Array.from({length: FACE_COUNT}, (_, face) => {
        const [a, b, c] = [0, 1, 2].map((k) => cornerOf(FACES[face * 3 + k]))
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

// The face a direction leaves the icosahedron through: every face is as far from the centre.
function faceOf(faces: readonly LatticeFace[], x: number, y: number, z: number): number {
    let face = 0
    let best = -Infinity
    for (let f = 0; f < faces.length; f++) {
        const [nx, ny, nz] = faces[f].normal
        const along = x * nx + y * ny + z * nz
        if (along > best) {
            best = along
            face = f
        }
    }
    return face
}

// The lattice point nearest a direction, as its canonical id, on the face's flat grid: a cell is a few thousandths of a radian wide.
export function cellOf(faces: readonly LatticeFace[], cols: number, x: number, y: number, z: number): number {
    const face = faceOf(faces, x, y, z)
    const {normal, height, origin, alongOf, downOf, ids} = faces[face]
    const reach = height / (x * normal[0] + y * normal[1] + z * normal[2])
    const q: Vec3 = [x * reach - origin[0], y * reach - origin[1], z * reach - origin[2]]
    const [j, i] = nearestOnGrid(dot(q, alongOf) * cols, dot(q, downOf) * cols)

    const down = Math.min(Math.max(i, 0), cols)
    const along = Math.min(Math.max(j, 0), cols - down)
    return canonicalId(cols, ids, down, along)
}

// The nearest point of a grid of equilateral triangles cut along their short diagonal: the corner with the largest weight.
export function nearestOnGrid(x: number, y: number): [number, number] {
    const fx = Math.floor(x), fy = Math.floor(y)
    const rx = x - fx, ry = y - fy
    if (rx + ry < 1) {
        const rest = 1 - rx - ry
        if (rest >= rx && rest >= ry) return [fx, fy]
        return rx >= ry ? [fx + 1, fy] : [fx, fy + 1]
    }
    const far = rx + ry - 1
    const right = 1 - ry, up = 1 - rx
    if (far >= right && far >= up) return [fx + 1, fy + 1]
    return right >= up ? [fx + 1, fy] : [fx, fy + 1]
}

// One pair per 32 canonical ids of a lattice of `cols`: which are tiles, and how many tiles come before them. A tile's
// texel is its place among the tiles in canonical order.
export type TileCells = {
    cols: number
    words: Uint32Array
    texels: Uint32Array
    // The inverse of texels: the 0-based tile at each texel.
    tiles: Uint32Array
}

export const IDS_PER_WORD = 32

// `ids` holds each tile's canonical id, in tile order.
export function cellsOf(ids: Uint32Array, cols: number): TileCells {
    const points = latticeSize(cols)
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
    return {cols, words, texels, tiles: byTexel}
}

// The tile whose cell holds a direction, by its 1-based id, or undefined over the sea.
export function tileAt(cells: TileCells, faces: readonly LatticeFace[], x: number, y: number, z: number): number | undefined {
    const id = cellOf(faces, cells.cols, x, y, z)
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

function cornerOf(corner: number): Vec3 {
    return [CORNERS[corner * 3], CORNERS[corner * 3 + 1], CORNERS[corner * 3 + 2]]
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
