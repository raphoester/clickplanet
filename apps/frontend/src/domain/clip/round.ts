import {Day, Story, THE_WORLD} from "./story.ts"

// What a day's clip plays: the run-up to its cutoff, the last hours of the fight for it.
export const ROUND_PLAY_MS = 3 * 3_600_000

// A stretch an arena's map played instead of the world's: the replay holds none of its takes. No close: it still plays.
export type Aside = {openedAt: number, closedAt: number | undefined}

// A day of the season, told from its winner by the run whose window holds its cutoff: twice a day, one run sees it. A
// battle fought in an arena is no story: its fighting is in none of the replay.
export function roundStoryOf(day: Day | undefined, since: number, until: number, asides: readonly Aside[]): Story | undefined {
    if (day === undefined || day.endedAt <= since || day.endedAt > until) return undefined
    if (asides.some(({openedAt, closedAt}) => openedAt < day.endedAt && (closedAt === undefined || day.endedAt <= closedAt))) return undefined
    const [winner, ...next] = day.podium
    if (winner === undefined) return undefined
    return {
        kind: "round", attacker: winner.country, rival: undefined, victims: next.map(({country}) => country),
        place: {region: THE_WORLD}, day,
    }
}
