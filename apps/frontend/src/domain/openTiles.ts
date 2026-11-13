import {Point} from "./clip/geometry.ts"
import {outcomeOf} from "./shields.ts"

export type OpenGround = {
    flag: string
    // x, y, z per tile, tile id - 1 first.
    positions: ArrayLike<number>
    ownerOf(tile: number): string | undefined
    shieldsOf(tile: number): number
    // While the reach plays, a click off the front is stopped.
    reach: boolean
}

type Lit = {nearestTo(point: Point, passes: (tile: number) => boolean): number | undefined}

// A tile one click takes: another flag's or free, with no shield.
export const opensOn = (ground: OpenGround) => (tile: number): boolean =>
    outcomeOf(ground.ownerOf(tile), ground.flag, ground.shieldsOf(tile)) === "taken"

// The open front tile nearest the point; for a flag with no land and no reach, the open tile nearest the point.
export function openTileNear(front: Lit, point: Point, ground: OpenGround): number | undefined {
    const open = opensOn(ground)
    const onFront = front.nearestTo(point, open)
    if (onFront !== undefined || ground.reach) return onFront

    const {positions} = ground
    let nearest: number | undefined
    let closest = -Infinity
    for (let tile = 1; tile * 3 <= positions.length; tile++) {
        if (ground.ownerOf(tile) === ground.flag) return undefined
        if (!open(tile)) continue
        const at = (tile - 1) * 3
        const along = positions[at] * point.x + positions[at + 1] * point.y + positions[at + 2] * point.z
        if (along > closest) {
            closest = along
            nearest = tile
        }
    }
    return nearest
}
