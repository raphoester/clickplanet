import {Beat, between, blend, clamp, framingOf, Shot, smooth, widestZoomOf} from "./camera.ts"
import {angleBetween, densest, dot, meanOf} from "./geometry.ts"

// A tile the flag took, and the continent it is on.
export type Visit = Beat & {region: string | undefined}

// A continent the tour stops at: framed on where the flag took the most there, and when it got there, as a share of
// the play.
export type Stop = {region: string, shot: Shot, arrival: number}

// A continent is a stop when this much of what the flag took is there.
const STOP_SHARE = 0.1

const MOST_STOPS = 4

// The flag got to a continent once a fifth of what it took there was taken.
const ARRIVAL = 0.2

// Where the flag took the most on a continent: Angola, not the whole of Africa from Guinea-Bissau to Mozambique, which
// only the whole globe frames.
const SPOT_RADIANS = 0.35

// The painted flags big enough to read, never down in the tiles.
const LEAST_STOP_ZOOM = 1.5

const MOST_STOP_ZOOM = 2.2

const HOLD_SECONDS = 0.3

const FLIGHT_SECONDS = 0.9

const STAY_SECONDS = 2.4

const REVEAL_SECONDS = 1.6

const END_HOLD_SECONDS = 1

// The camera closes in this much on a stop while it is there, so it never sits still.
const PUSH = 1.25

// Stops this far apart are flown between over the globe, from high up.
const FAR_RADIANS = 1

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
            }
        })
        .sort((a, b) => a.arrival - b.arrival)
}

// How long a tour of so many stops plays.
export function tourSecondsOf(stops: number): number {
    return HOLD_SECONDS + stops * (FLIGHT_SECONDS + STAY_SECONDS) + REVEAL_SECONDS + END_HOLD_SECONDS
}

// Opens on the whole globe over the first stop, flies from stop to stop in the order the flag got there, closing in
// on each, and pulls back out to the globe over all of them, which it holds before the call to act.
export function tourOf(stops: readonly Stop[], aspect: number, seconds: number): (share: number) => Shot {
    const globe = widestZoomOf(aspect)
    if (stops.length === 0) return () => ({direction: {x: 0, y: 0, z: 1}, zoom: globe})

    const start = {direction: stops[0].shot.direction, zoom: globe}
    const end = {direction: meanOf(stops.map(({shot}) => shot.direction)) ?? start.direction, zoom: globe}
    const flying = HOLD_SECONDS + stops.length * FLIGHT_SECONDS + REVEAL_SECONDS + END_HOLD_SECONDS
    const stay = Math.max(0.5, (seconds - flying) / stops.length)

    return (share) => {
        let at = clamp(share) * seconds - HOLD_SECONDS
        if (at < 0) return start
        let from: Shot = start
        for (const {shot} of stops) {
            if (at < FLIGHT_SECONDS) return flight(from, shot, at / FLIGHT_SECONDS, globe)
            at -= FLIGHT_SECONDS
            if (at < stay) return {direction: shot.direction, zoom: shot.zoom * PUSH ** (at / stay)}
            at -= stay
            from = {direction: shot.direction, zoom: shot.zoom * PUSH}
        }
        return at < REVEAL_SECONDS ? flight(from, end, at / REVEAL_SECONDS, globe) : end
    }
}

function flight(from: Shot, to: Shot, t: number, globe: number): Shot {
    const s = smooth(clamp(t))
    const high = clamp(angleBetween(from.direction, to.direction) / FAR_RADIANS) * Math.sin(Math.PI * s)
    return {direction: blend(from.direction, to.direction, s), zoom: between(between(from.zoom, to.zoom, s), globe, high)}
}
