import {TileChange} from "./changes.ts"
import {angleBetween, dot, densest, Point} from "./geometry.ts"

// About 1,300 km: a country and the ones around it.
const HEART_RADIANS = 0.2

// About 2,900 km: western Europe from France, and the neighbours a war spills into.
export const FRONT_RADIANS = 0.45

export type Front = {
    heart: Point
    changes: TileChange[]
}

// Two fronts this close, for one attacker, are one story told twice.
const SAME_FRONT_RADIANS = 0.6

export function sameFront(a: Front, b: Front): boolean {
    return angleBetween(a.heart, b.heart) < SAME_FRONT_RADIANS
}

// The window from the front's first change to its last: nothing still to wait through at the start.
export function spanOf(front: Front, margin: number): {since: number, until: number} {
    let since = Infinity
    let until = -Infinity
    for (const {at} of front.changes) {
        since = Math.min(since, at)
        until = Math.max(until, at)
    }
    return {since: since - margin, until: until + margin}
}

// Whether a tile is in a clip's sight: near its front, and on the country's ground when the clip is told on one. A
// bomb on the same country 6,000 km away would take the camera off the fighting.
export function inSightOf(
    front: Front,
    pointOf: (tile: number) => Point,
    groundOf: (tile: number) => string | undefined,
    scope: string | undefined,
): (tile: number) => boolean {
    const near = Math.cos(FRONT_RADIANS)
    return (tile) => dot(pointOf(tile), front.heart) >= near && (scope === undefined || groundOf(tile) === scope)
}

// Where most tiles changed hands, and every change around it.
export function frontOf(changes: readonly TileChange[], pointOf: (tile: number) => Point): Front | undefined {
    const heart = densest(changes.map(({tile}) => pointOf(tile)), HEART_RADIANS)
    if (heart === undefined) return undefined

    const near = Math.cos(FRONT_RADIANS)
    return {heart, changes: changes.filter(({tile}) => dot(pointOf(tile), heart) >= near)}
}
