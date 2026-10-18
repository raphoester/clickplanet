import {describe, expect, it} from "vitest"
import {roundStoryOf} from "./round.ts"
import {Day, storyKeyOf} from "./story.ts"

const CUTOFF = Date.parse("2026-10-17T21:00:00Z")

const day: Day = {
    season: 1, number: 5, finale: false, endedAt: CUTOFF,
    podium: [{country: "fr", rank: 1, points: 25}, {country: "pw", rank: 2, points: 18}, {country: "dz", rank: 3, points: 15}],
    table: [{country: "fr", rank: 1, points: 86}],
}

describe("a day of the season", () => {
    it("is told from its winner, the next places on its podium beside it, by the run whose window holds its cutoff", () => {
        const run = {since: Date.parse("2026-10-17T12:17:00Z"), until: Date.parse("2026-10-18T00:17:00Z")}

        expect(roundStoryOf(day, run.since, run.until)).toEqual({
            kind: "round", attacker: "fr", rival: undefined, victims: ["pw", "dz"], place: {region: "the world"}, taken: 25, day,
        })
    })

    it("is no story of a run that does not see its cutoff: the next one, twelve hours on, told it already", () => {
        expect(roundStoryOf(day, Date.parse("2026-10-18T00:17:00Z"), Date.parse("2026-10-18T12:17:00Z"))).toBeUndefined()
        expect(roundStoryOf(day, CUTOFF, CUTOFF + 3_600_000)).toBeUndefined()
    })

    it("is no story with nobody on its podium, or with no race", () => {
        expect(roundStoryOf({...day, podium: []}, CUTOFF - 1, CUTOFF)).toBeUndefined()
        expect(roundStoryOf(undefined, CUTOFF - 1, CUTOFF)).toBeUndefined()
    })

    it("is a story of its own: a flag winning day 5 then day 6 is two stories, told beside what the flag did", () => {
        const told = roundStoryOf(day, CUTOFF - 1, CUTOFF)!

        expect(storyKeyOf(told, () => "Europe")).toEqual({flag: "fr", continent: "the world", angle: "round", round: "1.5"})
        expect(storyKeyOf({...told, day: {...day, number: 6}}, () => "Europe").round).toBe("1.6")
    })
})
