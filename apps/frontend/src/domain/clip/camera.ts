import {angleBetween, densest, dot, meanOf, Point} from "./geometry.ts"

export type Shot = {
    direction: Point
    zoom: number
}

// Where a tile changed hands, and when in the clip's play, from 0 to 1.
export type Beat = {share: number, point: Point}

// A bomb: the shares of the clip that hold still on it, where it fell, and how close to see it.
export type Blast = {from: number, to: number, point: Point, zoom: number}

// When the camera comes back out of the tiles to the opening: never, for the last moments to show what changed, or
// halfway, so a steamroll's painted flags change on screen.
export type PullBack = "never" | "atEnd" | "midway"

// close: the zoom of the tiles. seconds: how long the replay plays.
export type Script = {
    close: number
    seconds: number
    pullBack: PullBack
    hold?: number
    blasts?: readonly Blast[]
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

// A bomb late in the clip pushes the pull back out later, down to this much of it.
const SHORTEST_REVEAL_SECONDS = 1.3

const MIDWAY = 0.5

const MIDWAY_SECONDS = 1.2

const BLAST_EASE = 0.05

// A blast is a fifth of the screen's width.
const BLAST_WIDTHS = 5

const GRID = 480

const SMOOTHING = 0.03

export function framingOf(points: readonly Point[], aspect: number): Shot {
    const least = aspect * WIDEST
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

export function blastZoomOf(radius: number, aspect: number): number {
    return Math.min(MAX_ZOOM, aspect / (BLAST_WIDTHS * Math.max(radius, 1e-3)))
}

// Opens on the opening shot, dives down into the tiles where they change hands and follows them, comes back out as
// pullBack says, and flies to every blast.
export function cameraOf(
    opening: Shot,
    beats: readonly Beat[],
    {close, seconds, pullBack, hold = HOLD_SECONDS, blasts = []}: Script,
): (share: number) => Shot {
    const keys = keysOf(opening.direction, beats)
    const out = outAt(pullBack, seconds, hold + DIVE_SECONDS, Math.max(0, ...blasts.map(({to}) => to * seconds)))
    const path = smoothed(Array.from({length: GRID + 1}, (_, i) => {
        const share = i / GRID
        const at = share * seconds
        const near = closenessAt(at, hold) * (pullBack === "midway" ? 1 - out(at) : 1)
        let shot = {direction: blend(opening.direction, keyAt(keys, share), near), zoom: between(opening.zoom, close, near)}
        for (const blast of blasts) {
            const pull = pullAt(blast, share)
            if (pull > 0) shot = {direction: blend(shot.direction, blast.point, pull), zoom: between(shot.zoom, Math.max(shot.zoom, blast.zoom), pull)}
        }
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

// 0 down in the tiles, 1 back out on the opening. At the end it waits for the last blast to be over.
function outAt(pullBack: PullBack, seconds: number, down: number, lastBlast: number): (at: number) => number {
    if (pullBack === "never") return () => 0
    const from = pullBack === "midway"
        ? Math.max(down, seconds * MIDWAY)
        : Math.max(down, Math.min(seconds - SHORTEST_REVEAL_SECONDS, Math.max(seconds - REVEAL_SECONDS, lastBlast)))
    const length = pullBack === "midway" ? MIDWAY_SECONDS : Math.max(1e-3, seconds - from)
    return (at) => smooth(clamp((at - from) / length))
}

// 0 on the opening shot, 1 down in the action.
function closenessAt(at: number, hold: number): number {
    return at < hold ? 0 : smooth(clamp((at - hold) / DIVE_SECONDS))
}

function pullAt({from, to}: Blast, share: number): number {
    if (share < from) return smooth(clamp(1 - (from - share) / BLAST_EASE))
    if (share > to) return smooth(clamp(1 - (share - to) / BLAST_EASE))
    return 1
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

function blend(a: Point, b: Point, t: number): Point {
    return meanOf([{x: a.x * (1 - t), y: a.y * (1 - t), z: a.z * (1 - t)}, {x: b.x * t, y: b.y * t, z: b.z * t}]) ?? a
}

function between(a: number, b: number, t: number): number {
    return a * (b / a) ** t
}

function clamp(value: number): number {
    return Math.min(1, Math.max(0, value))
}

function smooth(t: number): number {
    return t * t * (3 - 2 * t)
}
