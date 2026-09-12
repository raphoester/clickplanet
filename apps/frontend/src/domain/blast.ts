/**
 * A bomb dropped on the planet, as far as anything without a GPU cares.
 *
 * The server decides which tiles a bomb clears and says so; the client only
 * draws it. What lives here is the timing every screen has to agree on — when
 * the tiles go, and when the drawing is over — and `tilesWithin`, which the
 * fake backend uses to stand in for the server's own pick.
 */

/** How many blasts can be on screen at once. Matches `MAX_BLASTS` in the shader. */
export const MAX_BLASTS = 4

/**
 * The phases of one blast, in seconds. Mirrored as constants in the display
 * vertex shader: change one, change both.
 *
 * - `fall`: a ring closes in on the target while the bomb comes down. The
 *   tiles are still there — this is the warning.
 * - `shock`: the flash and the shock wave. The tiles are cleared at its start.
 * - `scorch`: the burnt ground cools back to the Earth underneath.
 */
export const BLAST_TIMELINE = {fall: 0.8, shock: 1.2, scorch: 5} as const

/** Seconds from the drop to the moment the tiles are cleared. */
export const IMPACT_DELAY = BLAST_TIMELINE.fall

/** What the news line says a bomb did, after the name of whoever dropped it. */
export function describeBlast(drop: {tile: number | undefined, cleared: readonly number[]}): string {
    if (drop.tile === undefined) return "bombed the ocean"
    if (drop.cleared.length === 0) return "bombed empty land"
    return `bombed ${drop.cleared.length} ${drop.cleared.length === 1 ? "tile" : "tiles"}`
}

/** Whether a blast started `elapsed` seconds ago has nothing left to draw. */
export function blastOver(elapsed: number): boolean {
    return elapsed >= BLAST_TIMELINE.fall + BLAST_TIMELINE.scorch
}

/**
 * Every tile whose centre is at most `radius` radians of arc from `center`,
 * `center` included. Tile ids are 1-based, positions are xyz triples.
 *
 * A straight scan: ~250k tiles in a millisecond or two, once per bomb.
 */
export function tilesWithin(positions: Float32Array, center: number, radius: number): number[] {
    const size = positions.length / 3
    if (!Number.isInteger(center) || center < 1 || center > size) return []

    const o = (center - 1) * 3
    const [cx, cy, cz] = unit(positions[o], positions[o + 1], positions[o + 2])
    const threshold = Math.cos(radius)

    const tiles: number[] = []
    for (let i = 0; i < size; i++) {
        const x = positions[i * 3], y = positions[i * 3 + 1], z = positions[i * 3 + 2]
        const length = Math.hypot(x, y, z)
        if (length === 0) continue
        if ((x * cx + y * cy + z * cz) / length >= threshold) tiles.push(i + 1)
    }
    return tiles
}

/**
 * The tile nearest to `target`, the arc to it in radians, and `target` put on
 * the unit sphere. `tile` is undefined for a target with no direction.
 */
export function nearestTile(
    positions: Float32Array,
    target: {x: number, y: number, z: number},
): {tile: number | undefined, arc: number, point: {x: number, y: number, z: number}} {
    const length = Math.hypot(target.x, target.y, target.z)
    if (!(length > 0) || !Number.isFinite(length)) return {tile: undefined, arc: Math.PI, point: {x: 0, y: 0, z: 1}}

    const [tx, ty, tz] = unit(target.x, target.y, target.z)

    let tile: number | undefined
    let best = -Infinity
    for (let i = 0; i < positions.length / 3; i++) {
        const x = positions[i * 3], y = positions[i * 3 + 1], z = positions[i * 3 + 2]
        const along = (x * tx + y * ty + z * tz) / (Math.hypot(x, y, z) || 1)
        if (along > best) {
            best = along
            tile = i + 1
        }
    }

    return {tile, arc: Math.acos(Math.min(best, 1)), point: {x: tx, y: ty, z: tz}}
}

function unit(x: number, y: number, z: number): [number, number, number] {
    const length = Math.hypot(x, y, z) || 1
    return [x / length, y / length, z / length]
}
