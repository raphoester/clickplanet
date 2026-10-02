// The geodesic lattice every tile is cut from, and how a point on it becomes a longitude and a
// latitude.
//
// The tiles are the land vertices of `THREE.IcosahedronGeometry(1, DETAIL)`, deduplicated by
// position. The dedup here must stay byte-identical to the one that produced the shipped blob —
// same rounding, same order — or the positions in an existing blob stop matching and every tile id
// moves. `lattice.test.mjs` pins the rule at a detail the suite can afford; `npm run map:audit`
// checks the whole thing against the blob on disk, where a mismatch would read as a quarter of a
// million tiles on open sea rather than as a few thousand.
import * as THREE from "three"

// Recovered from the shipped blob and then checked: scaling each tile's barycentric coordinates
// within its icosahedron face by `cols = DETAIL + 1` lands on integers with a worst error of 1e-5,
// float32's own precision, and gives errors around 0.49 — noise — at every other cols in 280..330.
// The backend re-runs that check on every boot and refuses to start if it fails.
export const DETAIL = 300

/** The key the dedup uses. Also what matches a lattice vertex to a position in an existing blob. */
export const keyOf = (x, y, z) => `${x.toFixed(6)},${y.toFixed(6)},${z.toFixed(6)}`

/**
 * Builds the whole sphere's vertices, deduplicated, in generation order.
 *
 * @param {number} detail
 * @returns {{count: number, positions: Float32Array, uvs: Float32Array}}
 */
export function lattice(detail = DETAIL) {
    const geometry = new THREE.IcosahedronGeometry(1, detail)
    const position = geometry.attributes.position.array
    const uv = geometry.attributes.uv.array

    const positions = []
    const uvs = []
    const seen = new Set()
    for (let i = 0; i < position.length; i += 3) {
        const key = keyOf(position[i], position[i + 1], position[i + 2])
        if (seen.has(key)) continue
        seen.add(key)
        positions.push(position[i], position[i + 1], position[i + 2])
        const u = (i / 3) * 2
        uvs.push(uv[u], uv[u + 1])
    }
    geometry.dispose()

    return {
        count: positions.length / 3,
        positions: Float32Array.from(positions),
        uvs: Float32Array.from(uvs),
    }
}

/**
 * Which lattice vertices touch which, read off the triangles they were generated from.
 *
 * Exact, and deliberately not a radius. `IcosahedronGeometry` subdivides each flat face and
 * normalises afterwards, so the spacing between touching vertices varies about 25% between a face's
 * middle and its corners: no single threshold separates a neighbour from the next ring out, and a
 * tuned one gave 2,137 tiles a degree of 7, 8 or even 10 when the backend tried it. The geometry is
 * non-indexed, so every triangle is three consecutive corners and its three sides are edges of the
 * lattice with nothing to infer.
 *
 * Returned as CSR — `at[i]` to `at[i + 1]` indexes `to` — so a walk allocates nothing per vertex.
 *
 * @param {number} detail
 * @returns {{count: number, at: Uint32Array, to: Uint32Array}}
 */
export function neighbours(detail = DETAIL) {
    const geometry = new THREE.IcosahedronGeometry(1, detail)
    const position = geometry.attributes.position.array

    const indexOf = new Map()
    const corners = new Uint32Array(position.length / 3)
    let count = 0
    for (let i = 0; i < position.length; i += 3) {
        const key = keyOf(position[i], position[i + 1], position[i + 2])
        let index = indexOf.get(key)
        if (index === undefined) indexOf.set(key, index = count++)
        corners[i / 3] = index
    }
    geometry.dispose()

    const degree = new Uint32Array(count + 1)
    for (let t = 0; t < corners.length; t += 3) {
        degree[corners[t]] += 2
        degree[corners[t + 1]] += 2
        degree[corners[t + 2]] += 2
    }

    const at = new Uint32Array(count + 1)
    for (let i = 0; i < count; i++) at[i + 1] = at[i] + degree[i]
    const filled = at.slice(0, count)
    const every = new Uint32Array(at[count])
    for (let t = 0; t < corners.length; t += 3) {
        const [a, b, c] = [corners[t], corners[t + 1], corners[t + 2]]
        every[filled[a]++] = b; every[filled[a]++] = c
        every[filled[b]++] = a; every[filled[b]++] = c
        every[filled[c]++] = a; every[filled[c]++] = b
    }

    // Each edge is walked once per triangle that carries it, so the same neighbour is written twice
    // over and three times at the twelve corners. Sorting each vertex's run and dropping the repeats
    // is what turns that back into the honeycomb.
    const to = new Uint32Array(at[count])
    const kept = new Uint32Array(count + 1)
    let out = 0
    for (let i = 0; i < count; i++) {
        kept[i] = out
        const run = every.subarray(at[i], at[i + 1])
        run.sort()
        let previous = -1
        for (const other of run) {
            if (other === previous || other === i) continue
            previous = other
            to[out++] = other
        }
    }
    kept[count] = out

    return {count, at: kept, to: to.subarray(0, out)}
}

// three's PolyhedronGeometry lays the uvs out equirectangularly, so this is a read of the same
// spherical coordinates the position carries rather than a second projection to keep in step:
// over all 257,948 shipped tiles the two disagree by at most 0.001 degrees. The globe's own texture
// is sampled with these uvs too, which is what makes the picture and the polygons comparable.
export const lonLatOf = (u, v) => [(u - 0.5) * 360, (v - 0.5) * 180]
