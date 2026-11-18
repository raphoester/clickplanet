import type {BorderLineData} from "./borderLinesFile.ts"

// An outline with fewer corners, for a view where a cell edge is under a pixel (the page's docs/country-outlines.md).
// lines: each run keeps its first and last corner and its two landmasses, and no corner moves.
// standsFor: two floats a piece, the cell edges it stands for and their chord over their length.
export type CoarseOutline = {lines: BorderLineData, standsFor: Float32Array}

// A piece's middle dips under the cell edges it stands for by at most this, a fifth of the over pass's lift above the
// tiles, so neither the tiles nor the earth hide it.
export const MOST_DIP = 1e-4
const MOST_CHORD = 2 * Math.sqrt(1 - (1 - MOST_DIP) ** 2)

// A piece is drawn wider by its slant, but one whose cell edges double back is drawn no wider than twice its line.
export const LEAST_SLANT = 0.5

// tolerance: how far a dropped corner may lie from the piece that replaces it, on the globe of radius 1.
export function coarseOutlinesOf(data: BorderLineData, tolerances: readonly number[]): CoarseOutline[] {
    const directions = directionsOf(data.corners)
    return tolerances.map((tolerance) => coarseOutlineOf(data, directions, tolerance))
}

function coarseOutlineOf(data: BorderLineData, directions: Float64Array, tolerance: number): CoarseOutline {
    const kept: number[] = []
    const standsFor: number[] = []
    const runs = new Uint32Array(data.runs.length)
    let first = 0
    data.runs.forEach((count, run) => {
        const last = first + count - 1
        const before = kept.length
        kept.push(first)
        // A ring keeps its corner farthest from where it starts, so it never shrinks to a point.
        const far = closed(data.corners, first, last) ? farthestOf(directions, first, last) : last
        for (const end of far === last ? [last] : [far, last]) {
            let at = kept[kept.length - 1]
            while (at < end) {
                const next = reachOf(directions, at, end, tolerance)
                standsFor.push(next - at, slantOf(directions, at, next))
                kept.push(next)
                at = next
            }
        }
        runs[run] = kept.length - before
        first = last + 1
    })

    const corners = new Int16Array(kept.length * 3)
    kept.forEach((corner, at) => corners.set(data.corners.subarray(corner * 3, corner * 3 + 3), at * 3))
    return {lines: {runs, sides: data.sides, corners}, standsFor: Float32Array.from(standsFor)}
}

// The furthest corner up to end that every corner between keeps within tolerance of, one step at least.
function reachOf(d: Float64Array, at: number, end: number, tolerance: number): number {
    const most = tolerance * tolerance
    const ax = d[at * 3], ay = d[at * 3 + 1], az = d[at * 3 + 2]
    let reach = at + 1
    for (let next = at + 2; next <= end; next++) {
        const bx = d[next * 3] - ax, by = d[next * 3 + 1] - ay, bz = d[next * 3 + 2] - az
        const length2 = bx * bx + by * by + bz * bz
        if (length2 > MOST_CHORD * MOST_CHORD) break
        for (let corner = at + 1; corner < next; corner++) {
            const px = d[corner * 3] - ax, py = d[corner * 3 + 1] - ay, pz = d[corner * 3 + 2] - az
            const t = Math.min(1, Math.max(0, (px * bx + py * by + pz * bz) / length2))
            const qx = px - t * bx, qy = py - t * by, qz = pz - t * bz
            if (qx * qx + qy * qy + qz * qz > most) return reach
        }
        reach = next
    }
    return reach
}

// From a corner to the piece between two others.
export function offsetOf(d: ArrayLike<number>, corner: number, from: number, to: number): number {
    const ax = d[from * 3], ay = d[from * 3 + 1], az = d[from * 3 + 2]
    const bx = d[to * 3] - ax, by = d[to * 3 + 1] - ay, bz = d[to * 3 + 2] - az
    const px = d[corner * 3] - ax, py = d[corner * 3 + 1] - ay, pz = d[corner * 3 + 2] - az
    const length2 = bx * bx + by * by + bz * bz
    const t = length2 > 0 ? Math.min(1, Math.max(0, (px * bx + py * by + pz * bz) / length2)) : 0
    return Math.hypot(px - t * bx, py - t * by, pz - t * bz)
}

function slantOf(directions: Float64Array, from: number, to: number): number {
    let length = 0
    for (let corner = from; corner < to; corner++) length += distanceOf(directions, corner, corner + 1)
    return length > 0 ? Math.max(LEAST_SLANT, distanceOf(directions, from, to) / length) : 1
}

function closed(corners: Int16Array, first: number, last: number): boolean {
    return last > first
        && corners[first * 3] === corners[last * 3]
        && corners[first * 3 + 1] === corners[last * 3 + 1]
        && corners[first * 3 + 2] === corners[last * 3 + 2]
}

function farthestOf(directions: Float64Array, first: number, last: number): number {
    let far = last, most = -1
    for (let corner = first + 1; corner < last; corner++) {
        const distance = distanceOf(directions, first, corner)
        if (distance > most) [far, most] = [corner, distance]
    }
    return far
}

function distanceOf(d: Float64Array, a: number, b: number): number {
    const x = d[b * 3] - d[a * 3], y = d[b * 3 + 1] - d[a * 3 + 1], z = d[b * 3 + 2] - d[a * 3 + 2]
    return Math.sqrt(x * x + y * y + z * z)
}

// As the vertex shader draws them: each corner on the globe of radius 1.
export function directionsOf(corners: Int16Array): Float64Array {
    const directions = new Float64Array(corners.length)
    for (let at = 0; at < corners.length; at += 3) {
        const length = Math.sqrt(corners[at] * corners[at] + corners[at + 1] * corners[at + 1] + corners[at + 2] * corners[at + 2])
        directions[at] = corners[at] / length
        directions[at + 1] = corners[at + 1] / length
        directions[at + 2] = corners[at + 2] / length
    }
    return directions
}
