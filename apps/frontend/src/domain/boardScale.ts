// What sizes a board's map: its spacing against earth-1's, which every count of tiles, length and zoom of the game was
// tuned on. The backend and the backoffice hold the same rule (docs/boards.md, "Rules at the new density").

// earth-1's manifest spacing, in radians: the reference.
export const EARTH_1_SPACING = 0.004063093462255102

// length: how much wider a tile is than on earth-1. density: how many more tiles cover the same ground.
export type BoardScale = {length: number, density: number}

export const EARTH_1_SCALE: BoardScale = scaleOf({spacing: EARTH_1_SPACING})

export function scaleOf(board: {spacing: number}): BoardScale {
    const length = board.spacing / EARTH_1_SPACING
    const across = EARTH_1_SPACING / board.spacing
    return {length, density: across * across}
}

// A count of tiles tuned on earth-1, as many on this board's ground: one at least.
export function scaledCount(n: number, scale: BoardScale): number {
    return Math.max(1, Math.round(n * scale.density))
}

// A length on the globe of radius 1 tuned on earth-1's tiles, as many of this board's tiles.
export function scaledLength(x: number, scale: BoardScale): number {
    return x * scale.length
}

// A camera zoom where a tile is as wide on screen as on earth-1.
export function scaledZoom(z: number, scale: BoardScale): number {
    return z / scale.length
}
