import {XpGain, XpSource} from "../backends/progression.ts"

// One gain on the level's line: what a source gave, with what the same source gave since it showed.
export type XpPop = {
    source: XpSource
    xp: number
    // One per pop, in the order they came.
    key: number
    // The gains added to it since it came: the line bumps at each one.
    bumps: number
}

// The pop on the line, and the ones waiting their turn, one per source.
export type XpPops = {
    shown?: XpPop
    waiting: XpPop[]
    next: number
}

export const NO_POPS: XpPops = {waiting: [], next: 1}

// A take is the common gain: its pop counts up while the player clicks, and lasts less once the clicks stop.
export const TILES_HOLD_MS = 1800
export const HOLD_MS = 3200

export function holdOf(pop: XpPop): number {
    return pop.source === "tilesTaken" ? TILES_HOLD_MS : HOLD_MS
}

// The gains of one fact, the tiles last: the day's first take shows its day before its tile. A source other than the
// tiles takes the line from the tiles at once, which wait their turn and keep counting: a player who keeps clicking
// would never let a box caught show.
export function poppedIn(pops: XpPops, gains: readonly XpGain[]): XpPops {
    const ordered = [...gains.filter((gain) => gain.source !== "tilesTaken"), ...gains.filter((gain) => gain.source === "tilesTaken")]
    return ordered.reduce(added, pops)
}

// The pop shown is done: the next one waiting shows.
export function poppedOut(pops: XpPops): XpPops {
    const [shown, ...waiting] = pops.waiting
    return {...pops, shown, waiting}
}

function added(pops: XpPops, gain: XpGain): XpPops {
    if (gain.xp <= 0) return pops
    if (pops.shown?.source === gain.source) return {...pops, shown: merged(pops.shown, gain)}

    const waiting = pops.waiting.findIndex((pop) => pop.source === gain.source)
    if (waiting !== -1) {
        return {...pops, waiting: pops.waiting.map((pop, i) => i === waiting ? merged(pop, gain) : pop)}
    }

    const pop: XpPop = {source: gain.source, xp: gain.xp, key: pops.next, bumps: 0}
    const next = pops.next + 1
    if (!pops.shown) return {...pops, shown: pop, next}
    if (pops.shown.source === "tilesTaken") return {shown: pop, waiting: [...pops.waiting, pops.shown], next}
    return {...pops, waiting: [...pops.waiting, pop], next}
}

function merged(pop: XpPop, gain: XpGain): XpPop {
    return {...pop, xp: pop.xp + gain.xp, bumps: pop.bumps + 1}
}
