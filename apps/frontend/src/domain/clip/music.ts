import {Story} from "./story.ts"

// The anthem a clip plays: the leading flag's, else the other side's, else the place's, else a victim's, for the
// flags with no recording.
export function anthemOf(story: Story, recorded: (country: string) => boolean): string | undefined {
    const place = "country" in story.place ? story.place.country : undefined
    return [story.attacker, story.rival, place, ...story.victims]
        .find((country): country is string => country !== undefined && recorded(country))
}
