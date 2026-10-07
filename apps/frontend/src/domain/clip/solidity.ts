import {angleBetween, Point} from "./geometry.ts"

// A tile on the lattice has 6 neighbours.
const NEIGHBOURS = 6

// Two tiles are neighbours up to this many spacings apart: past the first ring, short of the second.
const NEAR = 1.4

const SAMPLE = 200

// Below this, what was taken is lines drawn on someone else's land, not land taken.
export const SCRIBBLE_BELOW = 0.75

// For each tile taken, the share of its neighbours its taker holds at the end: about 1 for land taken, low for
// letters drawn on someone else's land. held is every tile the taker holds at the end; taken is part of it.
export function solidityOf(taken: readonly Point[], held: readonly Point[]): number {
    if (taken.length === 0 || held.length < 2) return 1
    const spacing = spacingOf(held)
    const neighbourhood = gridOf(held, spacing * NEAR)

    let solid = 0
    for (const point of taken) {
        const around = neighbourhood(point).filter((other) => {
            const angle = angleBetween(point, other)
            return angle > spacing / 2 && angle <= spacing * NEAR
        }).length
        solid += Math.min(1, around / NEIGHBOURS)
    }
    return solid / taken.length
}

// The usual distance from a tile to the nearest other one.
function spacingOf(points: readonly Point[]): number {
    const step = Math.max(1, Math.floor(points.length / SAMPLE))
    const nearest: number[] = []
    for (let i = 0; i < points.length; i += step) {
        let least = Infinity
        for (let j = 0; j < points.length; j++) {
            if (j !== i) least = Math.min(least, angleBetween(points[i], points[j]))
        }
        nearest.push(least)
    }
    nearest.sort((a, b) => a - b)
    return nearest[Math.floor(nearest.length / 2)]
}

// The points in the cubes of side cell around a point.
function gridOf(points: readonly Point[], cell: number): (point: Point) => Point[] {
    const keyOf = (x: number, y: number, z: number) => `${x},${y},${z}`
    const cells = new Map<string, Point[]>()
    for (const point of points) {
        const key = keyOf(Math.floor(point.x / cell), Math.floor(point.y / cell), Math.floor(point.z / cell))
        const inside = cells.get(key)
        if (inside) inside.push(point)
        else cells.set(key, [point])
    }
    return (point) => {
        const [x, y, z] = [point.x, point.y, point.z].map((value) => Math.floor(value / cell))
        const around: Point[] = []
        for (let i = -1; i <= 1; i++) for (let j = -1; j <= 1; j++) for (let k = -1; k <= 1; k++) {
            around.push(...cells.get(keyOf(x + i, y + j, z + k)) ?? [])
        }
        return around
    }
}
