import {Timestamp} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {
    Entry, GetScheduleResponse, GetSeasonsResponse, ScheduledFinale, ScheduledSeason, SeasonFinale,
} from "../gen/grpc/gamemaster/v1/admin_pb.ts"
import {Day, EntryKind, Season} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {GetRulesResponse, RuleSet} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Rules} from "../gen/grpc/planet/v1/rules_pb.ts"
import {placeOf, timelineOf, untilOf, upcoming} from "./timeline.ts"

const at = (iso: string) => Timestamp.fromDate(new Date(iso))

// Season 0's last day, its Final Battle, the freeze, and season 1's start on a blank map.
const night = {
    schedule: new GetScheduleResponse({
        entries: [new Entry({
            id: "freeze", ruleSet: "frozen", kind: EntryKind.PLAIN, priority: 1,
            startsAt: at("2026-10-31T23:00:00Z"), endsAt: at("2026-11-01T15:00:00Z"), madeAt: at("2026-10-28T10:00:00Z"),
        })],
        finales: [new ScheduledFinale({
            season: 0, finale: new SeasonFinale({ruleSet: "urf", times: 3}),
            startsAt: at("2026-10-31T21:00:00Z"), endsAt: at("2026-10-31T23:00:00Z"),
        })],
    }),
    seasons: new GetSeasonsResponse({seasons: [
        new ScheduledSeason({
            season: new Season({number: 0, startsAt: at("2026-10-08T00:00:00Z"), endsAt: at("2026-10-31T23:00:00Z")}),
            days: [new Day({number: 24, startsAt: at("2026-10-30T21:00:00Z"), endsAt: at("2026-10-31T21:00:00Z")})],
        }),
        new ScheduledSeason({
            season: new Season({number: 1, startsAt: at("2026-11-01T15:00:00Z"), endsAt: at("2026-11-28T23:00:00Z")}),
            days: [new Day({number: 1, startsAt: at("2026-11-01T15:00:00Z"), endsAt: at("2026-11-01T21:00:00Z")})],
        }),
    ]}),
    planet: new GetRulesResponse({sets: [
        new RuleSet({name: "default", rules: new Rules()}),
        new RuleSet({name: "frozen", rules: new Rules({frozen: true})}),
        new RuleSet({name: "urf", rules: new Rules()}),
    ]}),
}

const whatAt = (iso: string) =>
    timelineOf(night).instants.find(instant => instant.at.getTime() === new Date(iso).getTime())?.happenings.map(h => h.what)

describe("the timeline", () => {
    it("says what happens when the last day closes and the Final Battle starts, the day first", () => {
        expect(whatAt("2026-10-31T21:00:00Z")).toEqual(["Day 24 closes", "Final Battle ×3 starts (Season 0, urf)", "urf plays"])
    })

    it("says the Final Battle, the season and the open map end together, and the map freezes", () => {
        expect(whatAt("2026-10-31T23:00:00Z")).toEqual([
            "Final Battle ×3 ends", "Season 0 ends", "frozen plays: the map is frozen",
        ])
    })

    it("says season 1 starts on a blank map when the freeze ends", () => {
        expect(whatAt("2026-11-01T15:00:00Z")).toEqual(["Season 1 starts", "The map is reset", "default plays"])
    })

    it("draws what plays as blocks: the finale above all, then the freeze, with default left blank", () => {
        const rules = timelineOf(night).blocks.filter(block => block.lane === "rules")
        expect(rules.map(block => [block.label, block.tone, block.startsAt.toISOString(), block.endsAt.toISOString()])).toEqual([
            ["urf", "finale", "2026-10-31T21:00:00.000Z", "2026-10-31T23:00:00.000Z"],
            ["frozen", "frozen", "2026-10-31T23:00:00.000Z", "2026-11-01T15:00:00.000Z"],
        ])
    })

    it("puts the break between two seasons in the seasons lane", () => {
        const seasons = timelineOf(night).blocks.filter(block => block.lane === "seasons")
        expect(seasons.map(block => block.label)).toEqual(["Season 0", "Break", "Season 1"])
    })

    it("plays the entry of the highest priority, and at one priority the one made last", () => {
        const entry = (id: string, priority: number, madeAt: string) => new Entry({
            id, ruleSet: id, kind: EntryKind.PLAIN, priority, madeAt: at(madeAt),
            startsAt: at("2026-11-02T00:00:00Z"), endsAt: at("2026-11-02T06:00:00Z"),
        })
        const schedule = new GetScheduleResponse({entries: [
            entry("older", 2, "2026-10-01T00:00:00Z"), entry("newer", 2, "2026-10-02T00:00:00Z"), entry("lower", 1, "2026-10-03T00:00:00Z"),
        ]})
        const rules = timelineOf({...night, schedule}).blocks.filter(block => block.lane === "rules")
        expect(rules.map(block => block.label)).toEqual(["newer"])
    })

    it("names an entry by its kind", () => {
        const schedule = new GetScheduleResponse({entries: [new Entry({
            id: "spooky", ruleSet: "default", kind: EntryKind.HALLOWEEN, priority: 1, madeAt: at("2026-10-31T12:00:00Z"),
            startsAt: at("2026-11-02T00:00:00Z"), endsAt: at("2026-11-02T06:00:00Z"),
        })]})
        const rules = timelineOf({...night, schedule}).blocks.filter(block => block.lane === "rules")
        expect(rules.map(block => block.detail)).toEqual(["Halloween entry"])
    })

    it("keeps only what is still to come", () => {
        const instants = upcoming(timelineOf(night).instants, new Date("2026-10-31T22:00:00Z"))
        expect(instants[0].at.toISOString()).toBe("2026-10-31T23:00:00.000Z")
    })

    it("places a block along a range, clipped to it", () => {
        const from = new Date("2026-10-31T00:00:00Z")
        const to = new Date("2026-11-01T00:00:00Z")
        expect(placeOf({startsAt: new Date("2026-10-31T12:00:00Z"), endsAt: new Date("2026-11-02T00:00:00Z")}, from, to))
            .toEqual({left: 50, width: 50})
        expect(placeOf({startsAt: new Date("2026-11-02T00:00:00Z"), endsAt: new Date("2026-11-03T00:00:00Z")}, from, to)).toBeUndefined()
    })

    it("says how long until a time", () => {
        const now = new Date("2026-10-31T20:00:00Z")
        expect(untilOf(new Date("2026-10-31T21:00:00Z"), now)).toBe("in 1 h")
        expect(untilOf(new Date("2026-10-31T23:30:00Z"), now)).toBe("in 3 h 30 min")
        expect(untilOf(new Date("2026-11-01T22:00:00Z"), now)).toBe("in 1 d 2 h")
        expect(untilOf(new Date("2026-10-31T20:10:00Z"), now)).toBe("in 10 min")
        expect(untilOf(now, now)).toBe("now")
    })
})
