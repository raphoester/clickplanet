import {angleBetween, densest, dot, meanOf, Point} from "./geometry.ts"

export type Shot = {
    direction: Point
    zoom: number
}

// Where a tile changed hands, and when in the clip's play, from 0 to 1.
export type Beat = {share: number, point: Point}

// When the camera comes back out of the tiles to the opening, which every clip ends on: for its last moments, to show
// the map as it is now, or halfway, so a steamroll's painted flags change on screen.
export type PullBack = "atEnd" | "midway"

// close: the zoom of the tiles. seconds: how long the replay plays.
export type Script = {
    close: number
    seconds: number
    pullBack: PullBack
    hold?: number
}

const FILL = 0.6

const SPREAD_QUANTILE = 0.95

const MAX_ZOOM = 9

const WIDEST = 1.2

// About a continent: the opening is the map with its painted flags, however small the front.
const OPENING_ZOOM = 1.5

const KEYS = 24

// A key reads a twelfth of the clip and goes where the most tiles change hands in it.
const KEY_WINDOW = 1 / 12

const HEART_RADIANS = 0.1

const MOST_POINTS = 300

// The opening holds on the whole front, then the camera dives into the action.
const HOLD_SECONDS = 0.3

const DIVE_SECONDS = 0.7

const REVEAL_SECONDS = 1.6

// The view pulled back out holds this long before the call to act, while the last tiles change hands.
const END_HOLD_SECONDS = 1

const MIDWAY = 0.5

const MIDWAY_SECONDS = 1.2

// Up close the camera has to cross about this many screens a second, or it breathes: out to where the painted
// flags show and back into the tiles, every BREATH_SECONDS, halfway to the opening at most.
const BRISK_SCREENS_PER_SECOND = 0.4

const BREATH_SECONDS = 3

const BREATH_DEPTH = 0.5

// How long a stretch of the clip the camera's speed is read over.
const PACE_WINDOW = 0.05

const GRID = 480

const SMOOTHING = 0.03

// The whole globe, with a little sky around it.
export function widestZoomOf(aspect: number): number {
    return aspect * WIDEST
}

export function framingOf(points: readonly Point[], aspect: number): Shot {
    const least = widestZoomOf(aspect)
    const direction = meanOf(points)
    if (direction === undefined) return {direction: {x: 0, y: 0, z: 1}, zoom: least}

    const angles = points.map((point) => angleBetween(point, direction)).sort((a, b) => a - b)
    const spread = angles[Math.min(angles.length - 1, Math.floor(angles.length * SPREAD_QUANTILE))]
    const zoom = FILL * aspect / Math.max(Math.sin(Math.min(spread, Math.PI / 2)), 1e-3)

    return {direction, zoom: Math.min(MAX_ZOOM, Math.max(least, zoom))}
}

export function openingOf(framing: Shot): Shot {
    return {direction: framing.direction, zoom: Math.min(framing.zoom, OPENING_ZOOM)}
}

// How far the action moves up close, in screens: what a clip has to show beyond one place.
// Read on the camera's smoothed path, so a heart hopping between two places it never leaves counts for little.
export function screensOf(beats: readonly Beat[], close: number): number {
    const keys = keysOf(meanOf(beats.map(({point}) => point)) ?? {x: 0, y: 0, z: 1}, beats)
    const path = smoothed(Array.from({length: GRID + 1}, (_, i) => ({direction: keyAt(keys, i / GRID), zoom: close})))
    let travel = 0
    for (let i = 1; i < path.length; i++) travel += angleBetween(path[i - 1].direction, path[i].direction)
    return travel / screenOf(close)
}

// Opens on the opening shot, dives down into the tiles where they change hands and follows them, and comes back out
// as pullBack says. A bomb holds nothing still: the camera goes where the tiles change hands, and a bomb's crater is
// tiles changing hands.
export function cameraOf(
    opening: Shot,
    beats: readonly Beat[],
    {close, seconds, pullBack, hold = HOLD_SECONDS}: Script,
): (share: number) => Shot {
    const keys = keysOf(opening.direction, beats)
    const out = outAt(pullBack, seconds, hold + DIVE_SECONDS)
    const still = stillnessOf(keys, close, seconds)
    const path = smoothed(Array.from({length: GRID + 1}, (_, i) => {
        const share = i / GRID
        const at = share * seconds
        const breath = 1 - BREATH_DEPTH * still[i] * breathAt(at - hold - DIVE_SECONDS)
        const near = closenessAt(at, hold) * breath * (pullBack === "midway" ? 1 - out(at) : 1)
        const shot = {direction: blend(opening.direction, keyAt(keys, share), near), zoom: between(opening.zoom, close, near)}
        const end = pullBack === "atEnd" ? out(at) : 0
        return end > 0 ? {direction: blend(shot.direction, opening.direction, end), zoom: between(shot.zoom, opening.zoom, end)} : shot
    }))

    return (share) => {
        const position = clamp(share) * GRID
        const i = Math.min(GRID - 1, Math.floor(position))
        const t = position - i
        return {direction: blend(path[i].direction, path[i + 1].direction, t), zoom: between(path[i].zoom, path[i + 1].zoom, t)}
    }
}

// The height of the screen, in radians of the globe, at a zoom.
function screenOf(zoom: number): number {
    return 2 / zoom
}

// 1 where the camera would sit still up close, 0 where it already crosses the map briskly, along the grid.
function stillnessOf(keys: readonly Point[], close: number, seconds: number): number[] {
    const raw = Array.from({length: GRID + 1}, (_, i) => {
        const share = i / GRID
        const from = keyAt(keys, clamp(share - PACE_WINDOW / 2))
        const to = keyAt(keys, clamp(share + PACE_WINDOW / 2))
        const speed = angleBetween(from, to) / screenOf(close) / (PACE_WINDOW * Math.max(seconds, 1e-3))
        return 1 - clamp(speed / BRISK_SCREENS_PER_SECOND)
    })
    const reach = Math.round(PACE_WINDOW * GRID)
    return raw.map((_, i) => {
        const around = raw.slice(Math.max(0, i - reach), i + reach + 1)
        return around.reduce((sum, value) => sum + value, 0) / around.length
    })
}

// 0 in the tiles, 1 out at the painted flags, breathing from the moment the dive is down.
function breathAt(since: number): number {
    return since <= 0 ? 0 : (1 - Math.cos(2 * Math.PI * since / BREATH_SECONDS)) / 2
}

function keysOf(fallback: Point, beats: readonly Beat[]): Point[] {
    let last = fallback
    return Array.from({length: KEYS}, (_, k) => {
        const middle = (k + 0.5) / KEYS
        const points = beats.filter(({share}) => Math.abs(share - middle) <= KEY_WINDOW / 2).map(({point}) => point)
        return last = heartOf(points) ?? last
    })
}

function heartOf(points: readonly Point[]): Point | undefined {
    const step = Math.max(1, Math.ceil(points.length / MOST_POINTS))
    const heart = densest(points.filter((_, i) => i % step === 0), HEART_RADIANS)
    if (heart === undefined) return undefined
    const near = Math.cos(HEART_RADIANS)
    return meanOf(points.filter((point) => dot(point, heart) >= near)) ?? heart
}

function keyAt(keys: readonly Point[], share: number): Point {
    const position = share * KEYS - 0.5
    const k = Math.max(0, Math.min(KEYS - 2, Math.floor(position)))
    return blend(keys[k], keys[k + 1], clamp(position - k))
}

// 0 down in the tiles, 1 back out on the opening.
function outAt(pullBack: PullBack, seconds: number, down: number): (at: number) => number {
    const out = seconds - END_HOLD_SECONDS
    const from = pullBack === "midway" ? Math.max(down, seconds * MIDWAY) : Math.max(down, out - REVEAL_SECONDS)
    const length = pullBack === "midway" ? MIDWAY_SECONDS : Math.max(1e-3, out - from)
    return (at) => smooth(clamp((at - from) / length))
}

// 0 on the opening shot, 1 down in the action.
function closenessAt(at: number, hold: number): number {
    return at < hold ? 0 : smooth(clamp((at - hold) / DIVE_SECONDS))
}

// The keys turn the camera at a corner each; the zoom is smooth already.
function smoothed(shots: readonly Shot[]): Shot[] {
    const sigma = SMOOTHING * GRID
    const reach = Math.ceil(3 * sigma)
    const weights = Array.from({length: 2 * reach + 1}, (_, j) => Math.exp(-(((j - reach) / sigma) ** 2) / 2))
    return shots.map((_, i) => {
        let x = 0, y = 0, z = 0
        for (let j = -reach; j <= reach; j++) {
            const {direction} = shots[Math.max(0, Math.min(shots.length - 1, i + j))]
            const weight = weights[j + reach]
            x += direction.x * weight
            y += direction.y * weight
            z += direction.z * weight
        }
        return {direction: meanOf([{x, y, z}]) ?? shots[i].direction, zoom: shots[i].zoom}
    })
}

export function blend(a: Point, b: Point, t: number): Point {
    return meanOf([{x: a.x * (1 - t), y: a.y * (1 - t), z: a.z * (1 - t)}, {x: b.x * t, y: b.y * t, z: b.z * t}]) ?? a
}

export function between(a: number, b: number, t: number): number {
    return a * (b / a) ** t
}

export function clamp(value: number): number {
    return Math.min(1, Math.max(0, value))
}

export function smooth(t: number): number {
    return t * t * (3 - 2 * t)
}
