import {ranked, tally} from "./changes.ts"
import {Story} from "./story.ts"

const LED_FROM = 3
const FELL_PAST = 10

// A rout of a flag in the world's top 3 when it opened, out of the top 10 when it ends: never one place lost, as
// "GENERATIONAL DOWNFALL OF CHINA" told China losing Finland on 2026-10-28.
export function downfallOf(story: Story, opening: ReadonlyMap<number, string>, end: ReadonlyMap<number, string>): boolean {
    if (story.kind !== "rout") return false
    const loser = story.victims[0]
    return worldRankOf(loser, opening) <= LED_FROM && worldRankOf(loser, end) > FELL_PAST
}

// From 1, by tiles held across the world; past the last for a flag holding none.
function worldRankOf(flag: string, owners: ReadonlyMap<number, string>): number {
    const flags = ranked(tally(owners.values()))
    const rank = flags.findIndex(([owner]) => owner === flag)
    return rank === -1 ? flags.length + 1 : rank + 1
}
