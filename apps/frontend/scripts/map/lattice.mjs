import * as THREE from "three"

export const DETAIL = 300

// Must match the dedup that built the shipped blob exactly, or every tile id moves.
export const keyOf = (x, y, z) => `${x.toFixed(6)},${y.toFixed(6)},${z.toFixed(6)}`

/**
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
 * @param {number} detail
 * @returns {{count: number, at: Uint32Array, to: Uint32Array}} CSR: `at[i]` to `at[i + 1]` indexes `to`
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

export const lonLatOf = (u, v) => [(u - 0.5) * 360, (v - 0.5) * 180]
