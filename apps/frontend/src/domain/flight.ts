import {between, blend, clamp, Shot, smooth} from "./clip/camera.ts"
import {dot, meanOf, Point} from "./clip/geometry.ts"

export const FLIGHT_SECONDS = 0.9

// The share of the screen's height and width a territory flown to fills, leaving room for the panels at its sides.
const FILL = 0.6

const NARROWEST = 1e-3

type Box = {right: Point, up: Point, left: number, rightmost: number, bottom: number, top: number}

// The territory centred and as big as the screen allows, its tiles never smaller than at `least` nor bigger than at `most`.
export function territoryShotOf(points: readonly Point[], aspect: number, least: number, most: number): Shot | undefined {
    const mean = meanOf(points)
    if (mean === undefined) return undefined

    const direction = centreOf(boxOf(points, mean), mean)
    const box = boxOf(points, direction)
    const halfWidth = Math.max(-box.left, box.rightmost, NARROWEST)
    const halfHeight = Math.max(-box.bottom, box.top, NARROWEST)
    const zoom = FILL * Math.min(1 / halfHeight, aspect / halfWidth)

    return {direction, zoom: Math.min(most, Math.max(least, zoom))}
}

export function flightAt(from: Shot, to: Shot, share: number): Shot {
    const t = smooth(clamp(share))
    return {direction: blend(from.direction, to.direction, t), zoom: between(from.zoom, to.zoom, t)}
}

// The screen's axes for a camera on `direction` looking at the centre, the north pole up, as three's lookAt makes them.
function boxOf(points: readonly Point[], direction: Point): Box {
    const right = normalized({x: direction.z, y: 0, z: -direction.x}) ?? {x: 1, y: 0, z: 0}
    const up = cross(direction, right)
    const box = {right, up, left: 0, rightmost: 0, bottom: 0, top: 0}
    for (const point of points) {
        if (dot(point, direction) <= 0) continue
        const across = dot(point, right)
        const along = dot(point, up)
        box.left = Math.min(box.left, across)
        box.rightmost = Math.max(box.rightmost, across)
        box.bottom = Math.min(box.bottom, along)
        box.top = Math.max(box.top, along)
    }
    return box
}

function centreOf({right, up, left, rightmost, bottom, top}: Box, direction: Point): Point {
    const across = (left + rightmost) / 2
    const along = (bottom + top) / 2
    const depth = Math.sqrt(Math.max(0, 1 - across * across - along * along))
    return normalized({
        x: right.x * across + up.x * along + direction.x * depth,
        y: right.y * across + up.y * along + direction.y * depth,
        z: right.z * across + up.z * along + direction.z * depth,
    }) ?? direction
}

function cross(a: Point, b: Point): Point {
    return {x: a.y * b.z - a.z * b.y, y: a.z * b.x - a.x * b.z, z: a.x * b.y - a.y * b.x}
}

function normalized(point: Point): Point | undefined {
    const length = Math.hypot(point.x, point.y, point.z)
    return length < 1e-9 ? undefined : {x: point.x / length, y: point.y / length, z: point.z / length}
}
