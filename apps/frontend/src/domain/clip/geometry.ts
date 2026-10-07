export type Point = {x: number, y: number, z: number}

const MAX_CANDIDATES = 1500

const CELL_DEGREES = 10

const LATITUDES = 180 / CELL_DEGREES

const LONGITUDES = 360 / CELL_DEGREES

export const CELLS = LATITUDES * LONGITUDES

// positions holds x, y, z per tile, tile id - 1 first.
export function pointOf(positions: ArrayLike<number>, tile: number): Point {
    return {x: positions[(tile - 1) * 3], y: positions[(tile - 1) * 3 + 1], z: positions[(tile - 1) * 3 + 2]}
}

export function dot(a: Point, b: Point): number {
    return a.x * b.x + a.y * b.y + a.z * b.z
}

export function angleBetween(a: Point, b: Point): number {
    return Math.acos(Math.min(1, Math.max(-1, dot(a, b))))
}

export function meanOf(points: readonly Point[]): Point | undefined {
    let x = 0, y = 0, z = 0
    for (const point of points) {
        x += point.x
        y += point.y
        z += point.z
    }
    const length = Math.hypot(x, y, z)
    return length === 0 ? undefined : {x: x / length, y: y / length, z: z / length}
}

// The point with the most others within radius: the middle of points spread over two continents is open sea.
export function densest(points: readonly Point[], radius: number): Point | undefined {
    const step = Math.max(1, Math.floor(points.length / MAX_CANDIDATES))
    const close = Math.cos(radius)
    let best: Point | undefined
    let most = -1
    for (let i = 0; i < points.length; i += step) {
        const candidate = points[i]
        let count = 0
        for (const point of points) {
            if (dot(candidate, point) >= close) count++
        }
        if (count > most) {
            most = count
            best = candidate
        }
    }
    return best
}

// A cell of a 10° grid of latitudes and longitudes, for counting where things happen.
export function cellOf({x, y, z}: Point): number {
    const latitude = Math.asin(Math.min(1, Math.max(-1, y))) * 180 / Math.PI
    const longitude = Math.atan2(x, z) * 180 / Math.PI
    const row = Math.min(LATITUDES - 1, Math.floor((latitude + 90) / CELL_DEGREES))
    const column = Math.min(LONGITUDES - 1, Math.floor((longitude + 180) / CELL_DEGREES))
    return row * LONGITUDES + column
}

// The cell and the eight around it, round the antimeridian and never past a pole.
export function aroundCell(cell: number): number[] {
    const row = Math.floor(cell / LONGITUDES)
    const column = cell % LONGITUDES
    const around: number[] = []
    for (let r = Math.max(0, row - 1); r <= Math.min(LATITUDES - 1, row + 1); r++) {
        for (let c = column - 1; c <= column + 1; c++) {
            around.push(r * LONGITUDES + (c + LONGITUDES) % LONGITUDES)
        }
    }
    return around
}
