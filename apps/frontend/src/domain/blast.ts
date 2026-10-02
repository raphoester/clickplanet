// MAX_BLASTS and BLAST_TIMELINE are mirrored in shaders/display/vertex.glsl.
export const MAX_BLASTS = 4

export const BLAST_TIMELINE = {fall: 0.8, shock: 1.2, scorch: 5} as const

export const IMPACT_DELAY = BLAST_TIMELINE.fall

export function describeBlast(drop: {tile?: number, cleared: number}, land?: string): string {
    if (drop.tile === undefined) return "bombed the ocean"
    if (land) return `bombed ${land}`
    if (drop.cleared === 0) return "bombed empty land"
    return `bombed ${drop.cleared} ${drop.cleared === 1 ? "tile" : "tiles"}`
}

export function blastOver(elapsed: number): boolean {
    return elapsed >= BLAST_TIMELINE.fall + BLAST_TIMELINE.scorch
}

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
