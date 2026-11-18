import {latticeIds} from "../../src/domain/lattice.ts"
import {neighbours, lattice} from "./lattice.mjs"

/**
 * What a carry-over carries, as docs/boards.md records it: the tiles carried and blank, the children per parent (inland:
 * a parent whose every lattice neighbour is land), the distance from each child to its parent in the old board's
 * spacings, and the blank tiles an old tile lies within 0.6 spacings of.
 * @param {{manifest: import("../../src/domain/board.ts").Board, count: number, ids: Uint32Array, points: Uint32Array,
 *   positions: Float32Array}} from
 * @param {{count: number, positions: Float32Array}} to
 * @param {Uint32Array} parents
 */
export function carryStats(from, to, parents) {
    const shape = from.manifest.lattice
    const spacing = from.manifest.spacing
    const children = new Uint32Array(from.count + 1)
    const distances = []
    let carried = 0
    for (let t = 0; t < to.count; t++) {
        const parent = parents[t]
        if (parent === 0) continue
        carried++
        children[parent]++
        distances.push(arc(to.positions, t, from.positions, parent - 1) / spacing)
    }
    distances.sort((a, b) => a - b)

    const {at, to: next} = neighbours(shape)
    const {positions: points} = lattice(shape)
    const ids = latticeIds(shape)
    const tileOfPoint = new Int32Array(ids.length).fill(-1)
    for (let t = 0; t < from.count; t++) tileOfPoint[from.points[t]] = t
    let inland = 0, inlandChildren = 0, orphans = 0
    const histogram = {}
    for (let t = 0; t < from.count; t++) {
        const count = children[t + 1]
        histogram[count] = (histogram[count] ?? 0) + 1
        if (count === 0) orphans++
        const point = from.points[t]
        let land = true
        for (let e = at[point]; e < at[point + 1]; e++) land &&= tileOfPoint[next[e]] >= 0
        if (land) {
            inland++
            inlandChildren += count
        }
    }

    // A blank tile an old tile lies within 0.6 spacings of: its nearest old lattice point is the sea, and a tile is
    // among that point's neighbours.
    let blankNear = 0
    const cellPoint = nearestPointOf(points)
    for (let t = 0; t < to.count; t++) {
        if (parents[t] !== 0) continue
        const point = cellPoint(to.positions, t)
        for (let e = at[point]; e < at[point + 1]; e++) {
            const tile = tileOfPoint[next[e]]
            if (tile >= 0 && arc(to.positions, t, from.positions, tile) / spacing <= 0.6) {
                blankNear++
                break
            }
        }
    }

    return {
        carried,
        blank: to.count - carried,
        orphans,
        meanChildren: carried / (from.count - orphans),
        inlandChildren: inlandChildren / inland,
        histogram,
        median: distances[Math.floor(distances.length / 2)],
        p999: distances[Math.floor(distances.length * 0.999)],
        most: distances[distances.length - 1],
        blankNear,
    }
}

function arc(a, i, b, j) {
    const chord = Math.hypot(a[i * 3] - b[j * 3], a[i * 3 + 1] - b[j * 3 + 1], a[i * 3 + 2] - b[j * 3 + 2])
    return 2 * Math.asin(Math.min(1, chord / 2))
}

// The lattice point nearest a tile, by a grid of buckets on the cube: only for the stats, which run once.
function nearestPointOf(points) {
    const count = points.length / 3
    const cells = 256
    const buckets = new Map()
    const keyOf = (x, y, z) => `${Math.floor((x + 1) * cells / 2)},${Math.floor((y + 1) * cells / 2)},${Math.floor((z + 1) * cells / 2)}`
    for (let p = 0; p < count; p++) {
        const key = keyOf(points[p * 3], points[p * 3 + 1], points[p * 3 + 2])
        let bucket = buckets.get(key)
        if (bucket === undefined) buckets.set(key, bucket = [])
        bucket.push(p)
    }
    return (positions, t) => {
        const x = positions[t * 3], y = positions[t * 3 + 1], z = positions[t * 3 + 2]
        const bx = Math.floor((x + 1) * cells / 2), by = Math.floor((y + 1) * cells / 2), bz = Math.floor((z + 1) * cells / 2)
        let best = -1, bestD = Infinity
        for (let dx = -1; dx <= 1; dx++) for (let dy = -1; dy <= 1; dy++) for (let dz = -1; dz <= 1; dz++) {
            for (const p of buckets.get(`${bx + dx},${by + dy},${bz + dz}`) ?? []) {
                const d = (points[p * 3] - x) ** 2 + (points[p * 3 + 1] - y) ** 2 + (points[p * 3 + 2] - z) ** 2
                if (d < bestD) {
                    bestD = d
                    best = p
                }
            }
        }
        return best
    }
}
