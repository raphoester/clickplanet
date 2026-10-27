import {ClosedRound, RoundStanding} from "../backends/standings.ts"

export const REVEAL_WITHIN_MS = 6 * 60 * 60 * 1000
export const REVEAL_SEEN_STORAGE_KEY = "clickplanet-round-seen"
export const REVEAL_ROWS = 10
const PODIUM = 3

export const ROUND_REVEAL = {podium: 3.4, slide: 0.7, settle: 1.6, hold: 2.4, champion: 2.2} as const

export function roundOverLine(round: {number: number, finale: boolean, own: boolean}): string {
    if (!round.own) return `Day ${round.number} is over!`
    return round.finale ? "The Final Battle is over!" : "The battle is over!"
}

export function seasonOverLine(season: number): string {
    return `Season ${season} is over!`
}

export const MEDALS = ["🥇", "🥈", "🥉"] as const

export function medalOf(rank: number): string {
    return MEDALS[rank - 1] ?? ""
}

// round: a day's reveal. brief: a round of its own's. results: the season's table and its champion, after the brief of
// the round that ended the season.
export type Reveal = {kind: "round" | "brief" | "results", closed: ClosedRound}

export function revealsOf(closed: ClosedRound): Reveal[] {
    if (!closed.own) return [{kind: "round", closed}]
    if (!closed.finale) return [{kind: "brief", closed}]
    return [{kind: "brief", closed}, {kind: "results", closed}]
}

export function revealKeyOf(reveal: Reveal): string {
    const {season, number, endedAt} = reveal.closed
    switch (reveal.kind) {
        case "round":
            return `${season}:${number}:${endedAt}`
        case "brief":
            return `${season}:own:${number}:${endedAt}`
        case "results":
            return `${season}:own:${number}:${endedAt}:results`
    }
}

// The reveals after the one closed last.
export function revealsDue(closed: ClosedRound | undefined, now: number, seen: string | null): Reveal[] {
    if (!closed || now - closed.endedAt >= REVEAL_WITHIN_MS) return []
    const reveals = revealsOf(closed)
    return reveals.slice(reveals.findIndex((reveal) => revealKeyOf(reveal) === seen) + 1)
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
