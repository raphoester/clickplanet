// Splits each country's tiles into the separate pieces of land they actually form, and works out
// the frame a flag is painted in over each piece.
//
// A flag belongs to a piece of ground, not to a sovereignty. France is mainland *and* New Caledonia;
// one frame spanning both would stretch the tricolour across half the planet and paint nothing
// recognisable in either place.
//
// **Two tiles are the same piece when they touch on the lattice and carry the same country.** That
// used to be "when they are within 1.35 times the median spacing", which is the tuned radius the
// backend's own geography notes warn against: the spacing varies about 25% between a face's middle
// and its corners, so one threshold is too wide in one place and too narrow in another. Too wide is
// what matters here — it reaches past the neighbours and joins two islands across a strait into one
// landmass, which is a flag painted over open water between them.
import {SEA} from "./ground.mjs"

/**
 * @param {{
 *   count: number,
 *   positions: Float32Array,
 *   grounds: string[],
 *   tileOf: Int32Array,
 *   at: Uint32Array,
 *   to: Uint32Array,
 * }} map the tiles, the country under each, and the lattice they sit on
 * @returns {{codes: string[], assignment: Uint16Array, frames: Float32Array, totals: Uint32Array}}
 */
export function landmasses({count, positions, grounds, tileOf, at, to}) {
    const vertexOf = new Uint32Array(count)
    for (let v = 0; v < tileOf.length; v++) if (tileOf[v] >= 0) vertexOf[tileOf[v]] = v

    // Union-find over the lattice edges between two tiles of one country.
    const parent = new Int32Array(count)
    for (let t = 0; t < count; t++) parent[t] = t
    const find = (x) => {
        while (parent[x] !== x) x = parent[x] = parent[parent[x]]
        return x
    }
    for (let t = 0; t < count; t++) {
        const v = vertexOf[t]
        for (let e = at[v]; e < at[v + 1]; e++) {
            const other = tileOf[to[e]]
            if (other <= t) continue
            if (grounds[other] !== grounds[t]) continue
            const [a, b] = [find(t), find(other)]
            if (a !== b) parent[a] = b
        }
    }

    // Landmass 0 is no country. Nothing reaches it now that every tile has one, but the blob's
    // format keeps the slot and the frontend still reads it as "unclaimed ground".
    const codes = [SEA]
    const slotOf = new Map()
    const assignment = new Uint16Array(count)
    for (let t = 0; t < count; t++) {
        if (grounds[t] === SEA) continue
        const root = find(t)
        let slot = slotOf.get(root)
        if (slot === undefined) {
            slot = codes.length
            if (slot > 65535) throw new Error("more landmasses than a Uint16 can name")
            codes.push(grounds[t])
            slotOf.set(root, slot)
        }
        assignment[t] = slot
    }

    const members = Array.from({length: codes.length}, () => [])
    for (let t = 0; t < count; t++) if (assignment[t]) members[assignment[t]].push(t)

    const totals = new Uint32Array(codes.length)
    for (let k = 1; k < codes.length; k++) totals[k] = members[k].length

    return {codes, assignment, frames: framesOf(positions, codes, members), totals}
}

// A frame per landmass, for painting a flag across it: the centre direction, the east axis there,
// and how far the piece reaches along each axis measured over the surface. Neither borders nor tiles
// move, so this is static.
function framesOf(positions, codes, members) {
    const frames = new Float32Array(codes.length * 5)

    for (let k = 1; k < codes.length; k++) {
        const own = members[k]
        if (own.length === 0) continue

        let [mx, my, mz] = [0, 0, 0]
        for (const t of own) {
            mx += positions[t * 3]
            my += positions[t * 3 + 1]
            mz += positions[t * 3 + 2]
        }
        const length = Math.hypot(mx, my, mz) || 1
        let centre = [mx / length, my / length, mz / length]

        // The mean direction of a piece curved over a sphere can fall outside it, so snap to the
        // piece's own tile nearest that direction.
        let best = own[0]
        let bestDot = -2
        for (const t of own) {
            const dot = positions[t * 3] * centre[0]
                + positions[t * 3 + 1] * centre[1]
                + positions[t * 3 + 2] * centre[2]
            if (dot > bestDot) {
                bestDot = dot
                best = t
            }
        }
        centre = [positions[best * 3], positions[best * 3 + 1], positions[best * 3 + 2]]
        const unit = Math.hypot(...centre) || 1
        centre = centre.map((v) => v / unit)

        const flat = Math.hypot(centre[0], centre[2])
        const east = flat < 1e-4 ? [1, 0, 0] : [centre[2] / flat, 0, -centre[0] / flat]
        const north = [
            centre[1] * east[2] - centre[2] * east[1],
            centre[2] * east[0] - centre[0] * east[2],
            centre[0] * east[1] - centre[1] * east[0],
        ]

        let halfU = 0
        let halfV = 0
        for (const t of own) {
            const p = [positions[t * 3], positions[t * 3 + 1], positions[t * 3 + 2]]
            const along = p[0] * centre[0] + p[1] * centre[1] + p[2] * centre[2]
            if (along <= 0) continue
            const angle = Math.acos(Math.min(1, along))
            const u = p[0] * east[0] + p[1] * east[1] + p[2] * east[2]
            const v = p[0] * north[0] + p[1] * north[1] + p[2] * north[2]
            const spread = Math.hypot(u, v) || 1
            halfU = Math.max(halfU, Math.abs(angle * u / spread))
            halfV = Math.max(halfV, Math.abs(angle * v / spread))
        }

        frames[k * 5] = centre[0]
        frames[k * 5 + 1] = centre[1]
        frames[k * 5 + 2] = centre[2]
        frames[k * 5 + 3] = halfU
        frames[k * 5 + 4] = halfV
    }

    return frames
}
