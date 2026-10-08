import {flagOfContinent} from "../regions.ts"
import {Story} from "./story.ts"

// The anthem a clip plays is the one of whoever makes the moves: the attacker's, the other side's in a battle, or
// the continent's when a continent strikes back together. Never a loser's, but for a flag thrown out by many flags,
// none of which leads: the clip names no other and calls to fight for it. A clip with no such anthem plays none.
export function anthemOf(story: Story, recorded: (code: string) => boolean): string | undefined {
    const told = story.team !== undefined ? [flagOfContinent(story.team)]
        : story.kind === "rout" ? [story.victims[0]]
        : [story.attacker, story.rival]
    return told.find((code): code is string => code !== undefined && recorded(code))
}
