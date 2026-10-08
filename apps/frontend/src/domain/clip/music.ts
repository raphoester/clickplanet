import {flagOfContinent} from "../regions.ts"
import {Story} from "./story.ts"

// The anthem a clip plays is the one of whoever makes the moves: the attacker's, the other side's in a battle, or
// the continent's when a continent strikes back together. Never a loser's: a clip with no such anthem plays none.
export function anthemOf(story: Story, recorded: (code: string) => boolean): string | undefined {
    const movers = story.team !== undefined ? [flagOfContinent(story.team)] : [story.attacker, story.rival]
    return movers.find((code): code is string => code !== undefined && recorded(code))
}
