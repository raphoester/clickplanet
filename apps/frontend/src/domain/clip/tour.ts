import {Beat, between, blend, clamp, framingOf, heartOf, Shot, smooth, smoothed, widestZoomOf} from "./camera.ts"
import {angleBetween, densest, dot, meanOf} from "./geometry.ts"

// A tile the flag took, and the continent it is on.
export type Visit = Beat & {region: string | undefined}

// A continent the tour stops at: framed on where the flag took the most there, when it got there, as a share of the
// play, and every tile it took there, which the camera follows while it stays.
export type Stop = {region: string, shot: Shot, arrival: number, beats: readonly Beat[]}

// A continent is a stop when this much of what the flag took is there: Christmas Island took 6% of its day in Oceania,
// next to its own home, and that is part of its tour.
const STOP_SHARE = 0.05

const MOST_STOPS = 4

// The flag got to a continent once a fifth of what it took there was taken.
const ARRIVAL = 0.2

// Where the flag took the most on a continent: Angola, not the whole of Africa from Guinea-Bissau to Mozambique, which
// only the whole globe frames.
const SPOT_RADIANS = 0.35

// The painted flags big enough to read.
const LEAST_STOP_ZOOM = 1.5

const MOST_STOP_ZOOM = 2.2

// At a stop the camera is this much closer than its framing, never past where the painted flags start to blend into the
// tiles, blurred: the flags changing hands are what a tour shows.
const DIVE = 2

const HOLD_SECONDS = 0.3

const FLIGHT_SECONDS = 1.2

const STAY_SECONDS = 2.6

const REVEAL_SECONDS = 1.6

const END_HOLD_SECONDS = 1

// Between two stops far apart the camera rises this much over the lower of them, never to the whole globe: out to
// the globe and back in under a second was brutal.
const RISE = 1.6

const FAR_RADIANS = 1

// A key reads this much of the play around it, and goes where the flag takes the most there.
const KEY_WINDOW = 0.08

const GRID = 480

// The continents a flag took land on at once, in the order it got there.
export function stopsOf(visits: readonly Visit[], aspect: number): Stop[] {
    const byRegion = new Map<string, Visit[]>()
    for (const visit of visits) {
        if (visit.region === undefined) continue
        const there = byRegion.get(visit.region) ?? []
        there.push(visit)
        byRegion.set(visit.region, there)
    }
    return [...byRegion]
        .filter(([, there]) => there.length >= visits.length * STOP_SHARE)
        .sort((a, b) => b[1].length - a[1].length)
        .slice(0, MOST_STOPS)
        .map(([region, there]) => {
            const heart = densest(there.map(({point}) => point), SPOT_RADIANS) ?? there[0].point
            const spot = there.filter(({point}) => dot(point, heart) >= Math.cos(SPOT_RADIANS))
            const shares = spot.map(({share}) => share).sort((a, b) => a - b)
            const framing = framingOf(spot.map(({point}) => point), aspect)
            return {
                region,
                shot: {direction: framing.direction, zoom: Math.min(MOST_STOP_ZOOM, Math.max(LEAST_STOP_ZOOM, framing.zoom))},
                arrival: shares[Math.floor((shares.length - 1) * ARRIVAL)],
                beats: there.map(({share, point}) => ({share, point})),
            }
        })
        .sort((a, b) => a.arrival - b.arrival)
}

// How long a tour of so many stops plays.
export function tourSecondsOf(stops: number): number {
    return HOLD_SECONDS + stops * (FLIGHT_SECONDS + STAY_SECONDS) + REVEAL_SECONDS + END_HOLD_SECONDS
}

// Opens on the whole globe over the first stop and dives into it. At each stop it follows the flag's fighting there as
// it happens, then flies to the next, in the order the flag got there, rising a little over a long way: no zooming out
// and back in at every stop. It ends pulled back out to the globe over all of them, which it holds before the call to
// act. closest: the zoom the painted flags start to blend into the tiles at.
export function tourOf(stops: readonly Stop[], aspect: number, seconds: number, closest: number): (share: number) => Shot {
    const globe = widestZoomOf(aspect)
    if (stops.length === 0) return () => ({direction: {x: 0, y: 0, z: 1}, zoom: globe})

    const start = {direction: stops[0].shot.direction, zoom: globe}
    const end = {direction: meanOf(stops.map(({shot}) => shot.direction)) ?? start.direction, zoom: globe}
    const flying = HOLD_SECONDS + stops.length * FLIGHT_SECONDS + REVEAL_SECONDS + END_HOLD_SECONDS
    const stay = Math.max(0.5, (seconds - flying) / stops.length)
    const follow = stops.map(({shot, beats}) => followerOf(shot.direction, beats))

    const shotAt = (share: number): Shot => {
        let at = share * seconds - HOLD_SECONDS
        if (at < 0) return start
        let from: Shot = start
        for (const [k, {shot}] of stops.entries()) {
            const close = {direction: follow[k](share), zoom: Math.min(closest, shot.zoom * DIVE)}
            if (at < FLIGHT_SECONDS) return flight(from, close, at / FLIGHT_SECONDS, globe, k > 0)
            at -= FLIGHT_SECONDS
            if (at < stay) return close
            at -= stay
            from = close
        }
        return at < REVEAL_SECONDS ? flight(from, end, at / REVEAL_SECONDS, globe, false) : end
    }

    const path = smoothed(Array.from({length: GRID + 1}, (_, i) => shotAt(i / GRID)))
    return (share) => {
        const position = clamp(share) * GRID
        const i = Math.min(GRID - 1, Math.floor(position))
        const t = position - i
        return {direction: blend(path[i].direction, path[i + 1].direction, t), zoom: between(path[i].zoom, path[i + 1].zoom, t)}
    }
}

// Where the flag takes the most at a stop around a share of the play, else the last place it did, else the stop.
function followerOf(fallback: Shot["direction"], beats: readonly Beat[]): (share: number) => Shot["direction"] {
    const sorted = [...beats].sort((a, b) => a.share - b.share)
    let first = 0
    let last = fallback
    const keys = Array.from({length: GRID + 1}, (_, i) => {
        const middle = i / GRID
        while (first < sorted.length && sorted[first].share < middle - KEY_WINDOW / 2) first++
        let end = first
        while (end < sorted.length && sorted[end].share <= middle + KEY_WINDOW / 2) end++
        return last = heartOf(sorted.slice(first, end).map(({point}) => point)) ?? last
    })
    return (share) => keys[Math.round(clamp(share) * GRID)]
}

// rises: between two stops, high enough to see where it goes, never to the globe; from the globe in, or out to it,
// straight.
function flight(from: Shot, to: Shot, t: number, globe: number, rises: boolean): Shot {
    const s = smooth(clamp(t))
    const zoom = between(from.zoom, to.zoom, s)
    if (!rises) return {direction: blend(from.direction, to.direction, s), zoom}
    const high = Math.max(globe, Math.min(from.zoom, to.zoom) / RISE)
    const far = clamp(angleBetween(from.direction, to.direction) / FAR_RADIANS)
    return {direction: blend(from.direction, to.direction, s), zoom: between(zoom, Math.min(zoom, high), far * Math.sin(Math.PI * s))}
}
