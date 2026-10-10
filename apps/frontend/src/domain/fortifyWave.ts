// Must match MAX_FORTIFIES, FORTIFY_WAVE and FORTIFY_GLOW in shaders/display/vertex.glsl.
export const MAX_FORTIFY_WAVES = 4
export const FORTIFY_WAVE_SECONDS = 1.2
export const FORTIFY_GLOW_SECONDS = 0.7

export type WavePlan = {
    // In the order the front reaches them, and when, in seconds after it starts.
    tiles: Uint32Array
    arrivals: Float64Array
    // The angle from the closing tile to the farthest tile, in radians.
    reach: number
}

export function wavePlan(tiles: ArrayLike<number>, positions: ArrayLike<number>, closing: number): WavePlan {
    const cx = positions[(closing - 1) * 3], cy = positions[(closing - 1) * 3 + 1], cz = positions[(closing - 1) * 3 + 2]
    const centre = Math.hypot(cx, cy, cz) || 1

    const angles = new Float64Array(tiles.length)
    let reach = 0
    for (let i = 0; i < tiles.length; i++) {
        if (tiles[i] === closing) continue
        const at = (tiles[i] - 1) * 3
        const x = positions[at], y = positions[at + 1], z = positions[at + 2]
        const along = (x * cx + y * cy + z * cz) / ((Math.hypot(x, y, z) || 1) * centre)
        angles[i] = Math.acos(Math.min(1, Math.max(-1, along)))
        reach = Math.max(reach, angles[i])
    }

    const order = Array.from(angles.keys()).sort((a, b) => angles[a] - angles[b])
    return {
        tiles: Uint32Array.from(order, (i) => tiles[i]),
        arrivals: Float64Array.from(order, (i) => reach > 0 ? angles[i] / reach * FORTIFY_WAVE_SECONDS : 0),
        reach,
    }
}

// How many of the plan's tiles the front has reached after `elapsed` seconds.
export function reached(plan: WavePlan, elapsed: number): number {
    let low = 0, high = plan.arrivals.length
    while (low < high) {
        const mid = (low + high) >> 1
        if (plan.arrivals[mid] <= elapsed) low = mid + 1
        else high = mid
    }
    return low
}
