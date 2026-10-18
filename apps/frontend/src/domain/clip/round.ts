import {Day, Story, THE_WORLD} from "./story.ts"

// What a day's clip plays: the run-up to its cutoff, the last hours of the fight for it.
export const ROUND_PLAY_MS = 3 * 3_600_000

// A day of the season, told from its winner by the run whose window holds its cutoff: twice a day, one run sees it.
export function roundStoryOf(day: Day | undefined, since: number, until: number): Story | undefined {
    if (day === undefined || day.endedAt <= since || day.endedAt > until) return undefined
    const [winner, ...next] = day.podium
    if (winner === undefined) return undefined
    return {
        kind: "round", attacker: winner.country, rival: undefined, victims: next.map(({country}) => country),
        place: {region: THE_WORLD}, taken: winner.points, day,
    }
}
