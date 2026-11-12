import {describe, expect, it} from "vitest"
import {countdownsOf, endClock, timeLeftTo} from "./seasonClock.ts"

const SECOND = 1000
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR

const endsAt = Date.UTC(2026, 9, 31, 23)
const season = {number: 0, startsAt: Date.UTC(2026, 9, 3), endsAt, bookId: "season-0"}
const lastDayEndsAt = endsAt - 2 * HOUR

const day = {number: 21, startsAt: lastDayEndsAt - 4 * DAY, endsAt: lastDayEndsAt - 3 * DAY, roundId: "season-0/day-21"}

describe("countdownsOf", () => {
    it("counts down to today's end and to the season's", () => {
        expect(countdownsOf(season, day.endsAt - (13 * HOUR + 5 * MINUTE + 20 * SECOND), undefined, day.endsAt)).toEqual([
            {kind: "today", countdown: {days: 0, hours: 13, minutes: 5, seconds: 20}},
            {kind: "season", countdown: {days: 3, hours: 15, minutes: 5, seconds: 20}},
        ])
    })

    it("counts the season alone with no day running", () => {
        expect(countdownsOf(season, lastDayEndsAt)).toEqual([{kind: "season", countdown: {days: 0, hours: 2, minutes: 0, seconds: 0}}])
        expect(countdownsOf(season, lastDayEndsAt, undefined, lastDayEndsAt).map((c) => c.kind)).toEqual(["season"])
        expect(countdownsOf(season, endsAt - (12 * MINUTE + 5 * SECOND))).toEqual([
            {kind: "season", countdown: {days: 0, hours: 0, minutes: 12, seconds: 5}},
        ])
    })

    it("counts a day that ends with the season once", () => {
        expect(countdownsOf(season, endsAt - 3 * HOUR, undefined, endsAt)).toEqual([{kind: "season", countdown: {days: 0, hours: 3, minutes: 0, seconds: 0}}])
    })

    it("rounds a part second up, so it reads 00m 01s until the end", () => {
        expect(countdownsOf(season, endsAt - 300)[0].countdown).toEqual({days: 0, hours: 0, minutes: 0, seconds: 1})
    })

    it("is empty once the season is over", () => {
        expect(countdownsOf(season, endsAt)).toEqual([])
        expect(countdownsOf(season, endsAt + DAY)).toEqual([])
    })

    const battle = {entryId: "", startsAt: lastDayEndsAt, endsAt, description: "Triple points.", times: 3, endsSeason: true, roundId: "season-0/finale"}
    const raid = {...battle, entryId: "raid", startsAt: lastDayEndsAt - DAY + HOUR, endsAt: lastDayEndsAt - DAY + 2 * HOUR, endsSeason: false, roundId: "raid"}

    it("counts down to the next battle between today's end and the season's", () => {
        expect(countdownsOf(season, lastDayEndsAt - DAY - 3 * HOUR, raid, lastDayEndsAt - DAY)).toEqual([
            {kind: "today", countdown: {days: 0, hours: 3, minutes: 0, seconds: 0}},
            {kind: "battle", countdown: {days: 0, hours: 4, minutes: 0, seconds: 0}},
            {kind: "season", countdown: {days: 1, hours: 5, minutes: 0, seconds: 0}},
        ])
    })

    it("names the season's finale the final battle", () => {
        expect(countdownsOf(season, lastDayEndsAt - DAY - 3 * HOUR, battle, lastDayEndsAt - DAY)).toEqual([
            {kind: "today", countdown: {days: 0, hours: 3, minutes: 0, seconds: 0}},
            {kind: "final", countdown: {days: 1, hours: 3, minutes: 0, seconds: 0}},
            {kind: "season", countdown: {days: 1, hours: 5, minutes: 0, seconds: 0}},
        ])
    })

    it("drops the battle once it starts", () => {
        expect(countdownsOf(season, battle.startsAt - SECOND, battle, lastDayEndsAt).map((c) => c.kind)).toEqual(["today", "final", "season"])
        expect(countdownsOf(season, battle.startsAt, battle, lastDayEndsAt).map((c) => c.kind)).toEqual(["season"])
    })
})

describe("endClock", () => {
    it("counts down to the end of the season, unit by unit", () => {
        expect(endClock(season, endsAt - (25 * DAY + 3 * HOUR + 7 * MINUTE + 9 * SECOND))).toEqual({days: 25, hours: 3, minutes: 7, seconds: 9})
        expect(endClock(season, endsAt - 300)).toEqual({days: 0, hours: 0, minutes: 0, seconds: 1})
    })

    it("is gone once the season is over", () => {
        expect(endClock(season, endsAt)).toBeUndefined()
    })
})

describe("timeLeftTo", () => {
    it("counts down to a moment, and stops at zero", () => {
        expect(timeLeftTo(endsAt, endsAt - (HOUR + 2 * SECOND))).toEqual({days: 0, hours: 1, minutes: 0, seconds: 2})
        expect(timeLeftTo(endsAt, endsAt + MINUTE)).toEqual({days: 0, hours: 0, minutes: 0, seconds: 0})
    })
})
