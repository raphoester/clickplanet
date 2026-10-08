import {ClosedRound, RoundStanding} from "../backends/standings.ts"

export const REVEAL_WITHIN_MS = 6 * 60 * 60 * 1000
export const REVEAL_SEEN_STORAGE_KEY = "clickplanet-round-seen"
export const REVEAL_ROWS = 10
const PODIUM = 3

export const ROUND_REVEAL = {podium: 3.4, slide: 0.7, settle: 1.6} as const

export function roundOverLine(round: {number: number, finale: boolean}): string {
    return round.finale ? "The Final Battle is over!" : `Day ${round.number} is over!`
}

export const MEDALS = ["🥇", "🥈", "🥉"] as const

export function medalOf(rank: number): string {
    return MEDALS[rank - 1] ?? ""
}

export function revealKeyOf(closed: ClosedRound): string {
    return `${closed.season}:${closed.number}:${closed.endedAt}`
}

export function revealDue(closed: ClosedRound | undefined, now: number, seen: string | null): closed is ClosedRound {
    if (!closed) return false
    return now - closed.endedAt < REVEAL_WITHIN_MS && revealKeyOf(closed) !== seen
}

export function podiumOf(closed: ClosedRound): RoundStanding[] {
    return closed.standings.filter((standing) => standing.rank <= PODIUM && standing.points > 0).slice(0, PODIUM)
}

export type Placing = {rank: number, points: number}

export type RevealRow = {
    countryCode: string
    before?: Placing
    after: Placing
    gained: number
    from: number
    to: number
    mine: boolean
}

export function revealRowsOf(closed: ClosedRound, countryCode: string, shown = REVEAL_ROWS): RevealRow[] {
    const mine = closed.after.findIndex((score) => score.countryCode === countryCode)
    const kept = closed.after.filter((_, i) => i < shown || i === mine)
    const beforeAt = new Map(closed.before.map((score, i) => [score.countryCode, i]))
    const gained = new Map(closed.standings.map((standing) => [standing.countryCode, standing.points]))

    const byBefore = kept
        .map((score, to) => ({code: score.countryCode, to, was: beforeAt.get(score.countryCode) ?? Infinity}))
        .sort((a, b) => a.was - b.was || a.to - b.to)
    const from = new Map(byBefore.map((row, i) => [row.code, i]))

    return kept.map((score, to) => {
        const was = beforeAt.get(score.countryCode)
        const before = was === undefined ? undefined : closed.before[was]
        return {
            countryCode: score.countryCode,
            before: before && {rank: before.rank, points: before.points},
            after: {rank: score.rank, points: score.points},
            gained: gained.get(score.countryCode) ?? 0,
            from: from.get(score.countryCode) ?? to,
            to,
            mine: score.countryCode === countryCode,
        }
    })
}

export type Move = {kind: "new"} | {kind: "up" | "down", by: number} | {kind: "same"}

export function moveOf(row: RevealRow): Move {
    if (!row.before) return {kind: "new"}
    const by = row.before.rank - row.after.rank
    if (by > 0) return {kind: "up", by}
    if (by < 0) return {kind: "down", by: -by}
    return {kind: "same"}
}
