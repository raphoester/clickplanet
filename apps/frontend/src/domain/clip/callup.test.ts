import {describe, expect, it} from "vitest"
import {callUpsOf} from "./callup.ts"
import {Day, storyKeyOf} from "./story.ts"

const CUTOFF = Date.parse("2026-10-23T21:00:00Z")
const FINALE = Date.parse("2026-10-31T21:00:00Z")
const RUN = {since: Date.parse("2026-10-23T12:17:00Z"), until: Date.parse("2026-10-24T00:17:00Z")}

const day: Day = {
    season: 0, number: 15, finale: false, endedAt: CUTOFF,
    podium: [{country: "dz", rank: 1, points: 25}, {country: "pw", rank: 2, points: 18}, {country: "ps", rank: 3, points: 15}],
    table: [
        {country: "dz", rank: 1, points: 289},
        {country: "pl", rank: 2, points: 275},
        {country: "pt", rank: 3, points: 210},
        {country: "fr", rank: 4, points: 117},
    ],
}

const called = (told: ReturnType<typeof callUpsOf>) => told.map(({attacker, victims}) => [attacker, ...victims])

describe("a country called up to the Final Battle", () => {
    it("is each country that can still win the season, the leader beside it, and the leader against the next", () => {
        const told = callUpsOf(day, FINALE, RUN.since, RUN.until)

        expect(called(told)).toEqual([["dz", "pl"], ["pl", "dz"]])
        expect(told[1]).toEqual({
            kind: "callup", attacker: "pl", rival: undefined, victims: ["dz"], place: {region: "the world"},
            day, finaleAt: FINALE,
        })
    })

    it("is a country that winning every day left and the Final Battle, the leader second each time, brings level, and no leader nobody can catch", () => {
        const level = {...day, table: [{country: "dz", rank: 1, points: 289}, {country: "pt", rank: 2, points: 289 - 7 * (8 + 3)}]}
        const short = {...day, table: [{country: "dz", rank: 1, points: 289}, {country: "pt", rank: 2, points: 289 - 7 * (8 + 3) - 1}]}

        expect(called(callUpsOf(level, FINALE, RUN.since, RUN.until))).toEqual([["dz", "pt"], ["pt", "dz"]])
        expect(callUpsOf(short, FINALE, RUN.since, RUN.until)).toEqual([])
    })

    it("is fewer countries as the days run out", () => {
        const lastDay = {...day, endedAt: FINALE - 86_400_000, table: [...day.table, {country: "ie", rank: 2, points: 260}]}

        expect(called(callUpsOf(lastDay, FINALE, lastDay.endedAt - 1, lastDay.endedAt))).toEqual([["dz", "pl"], ["pl", "dz"]])
    })

    it("is told by the run whose window holds the day's cutoff, as the day is", () => {
        expect(callUpsOf(day, FINALE, Date.parse("2026-10-24T00:17:00Z"), Date.parse("2026-10-24T12:17:00Z"))).toEqual([])
        expect(callUpsOf(day, FINALE, CUTOFF, CUTOFF + 3_600_000)).toEqual([])
    })

    it("is nobody once the Final Battle started or the season ended, nor with no season or no race", () => {
        expect(callUpsOf({...day, endedAt: FINALE}, FINALE, FINALE - 1, FINALE)).toEqual([])
        expect(callUpsOf({...day, finale: true}, FINALE, RUN.since, RUN.until)).toEqual([])
        expect(callUpsOf(day, undefined, RUN.since, RUN.until)).toEqual([])
        expect(callUpsOf(undefined, FINALE, RUN.since, RUN.until)).toEqual([])
    })

    it("is nobody when the day scored nothing", () => {
        expect(callUpsOf({...day, podium: []}, FINALE, RUN.since, RUN.until)).toEqual([])
    })

    it("is a story of each day: Poland called up after day 15 and after day 16 is two stories", () => {
        const poland = callUpsOf(day, FINALE, RUN.since, RUN.until)[1]

        expect(storyKeyOf(poland, () => "Europe")).toEqual({flag: "pl", continent: "the world", angle: "callup", round: "0.15"})
        expect(storyKeyOf({...poland, day: {...day, number: 16}}, () => "Europe").round).toBe("0.16")
    })
})
