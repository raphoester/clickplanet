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
