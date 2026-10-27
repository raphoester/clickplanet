import {Day, Story, THE_WORLD} from "./story.ts"

const FINALE_TIMES = 3

const DAY_MS = 86_400_000

// A country contends when winning every day left and the Final Battle, the leader second each time, brings it level.
// Only a day calls up: a battle's podium scored its times.
export function callUpsOf(day: Day | undefined, finaleAt: number | undefined, since: number, until: number): Story[] {
    if (day === undefined || finaleAt === undefined || day.finale || day.own) return []
    if (day.endedAt <= since || day.endedAt > until || day.endedAt >= finaleAt) return []

    const [first, second] = [...day.podium].sort((a, b) => b.points - a.points).map(({points}) => points)
    const gain = (first ?? 0) - (second ?? 0)
    const left = gain * (Math.round((finaleAt - day.endedAt) / DAY_MS) + FINALE_TIMES)
    const table = [...day.table].sort((a, b) => b.points - a.points)
    const leader = table[0]
    if (leader === undefined || first === undefined) return []

    const contenders = table.filter(({points}) => points > 0 && points + left >= leader.points)
    if (contenders.length < 2) return []
    return contenders.map(({country}) => {
        const next = contenders.find((other) => other.country !== country)
        return {
            kind: "callup", attacker: country, rival: undefined,
            victims: [country === leader.country && next !== undefined ? next.country : leader.country],
            place: {region: THE_WORLD}, day, finaleAt,
        }
    })
}
