import {aroundCell, CELLS} from "./geometry.ts"

// A tile changing hands, and whether it was taken from another flag rather than off empty ground.
export type Moment = {at: number, cell: number, captured: boolean}

export type Window = {since: number, until: number}

// A stretch of time and the cell its action is around.
export type Candidate = Window & {cell: number}

const BUCKET_MS = 15 * 60_000

const HOUR_MS = 3_600_000

const LENGTHS_HOURS = [1, 2, 3, 4, 6, 8, 12, 24]

const PLACES_PER_LENGTH = 4

const AROUND = Array.from({length: CELLS}, (_, cell) => aroundCell(cell))

// For each length, the busiest stretch of a few places far apart, by tiles taken from another flag around them.
export function candidatesOf(moments: readonly Moment[], since: number, until: number, hours?: number): Candidate[] {
    const buckets = Math.max(1, Math.ceil((until - since) / BUCKET_MS))
    const counts = new Int32Array(buckets * CELLS)
    for (const {at, cell, captured} of moments) {
        if (!captured || at < since || at > until) continue
        counts[Math.min(buckets - 1, Math.floor((at - since) / BUCKET_MS)) * CELLS + cell]++
    }

    const lengths = (hours === undefined ? LENGTHS_HOURS : [hours])
        .map((length) => Math.min(buckets, Math.max(1, Math.round(length * HOUR_MS / BUCKET_MS))))
        .filter((length, i, all) => all.indexOf(length) === i)

    return lengths.flatMap((length) => placesOfLength(counts, buckets, length).map(({cell, start}) => ({
        cell,
        since: since + start * BUCKET_MS,
        until: Math.min(until, since + (start + length) * BUCKET_MS),
    })))
}

function placesOfLength(counts: Int32Array, buckets: number, length: number): {cell: number, start: number}[] {
    const sums = new Int32Array(CELLS)
    const best = Array.from({length: CELLS}, () => ({start: 0, action: 0}))
    for (let bucket = 0; bucket < length; bucket++) add(sums, counts, bucket, 1)

    for (let start = 0; start + length <= buckets; start++) {
        if (start > 0) {
            add(sums, counts, start - 1, -1)
            add(sums, counts, start + length - 1, 1)
        }
        for (let cell = 0; cell < CELLS; cell++) {
            if (sums[cell] === 0) continue
            let action = 0
            for (const near of AROUND[cell]) action += sums[near]
            if (action > best[cell].action) best[cell] = {start, action}
        }
    }

    const chosen: {cell: number, start: number}[] = []
    const taken = new Set<number>()
    const byAction = best.map((place, cell) => ({cell, ...place})).filter(({action}) => action > 0)
        .sort((a, b) => b.action - a.action || a.cell - b.cell)
    for (const {cell, start} of byAction) {
        if (chosen.length >= PLACES_PER_LENGTH) break
        if (AROUND[cell].some((near) => taken.has(near))) continue
        chosen.push({cell, start})
        for (const near of AROUND[cell]) taken.add(near)
    }
    return chosen
}

function add(sums: Int32Array, counts: Int32Array, bucket: number, sign: number) {
    for (let cell = 0; cell < CELLS; cell++) sums[cell] += sign * counts[bucket * CELLS + cell]
}

export function inCandidate(candidate: Candidate, cell: number): boolean {
    return AROUND[candidate.cell].includes(cell)
}
