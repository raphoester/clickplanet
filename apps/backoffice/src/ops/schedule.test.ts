import {Timestamp} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {GetScheduleResponse, Playing} from "../gen/grpc/seasons/v1/admin_pb.ts"
import {changeable, complete, draftOf, lacking, playable, playingOf, rowsOf} from "./schedule.ts"

describe("the schedule's rows", () => {
    it("come by start, the first season's days first", () => {
        const rows = rowsOf(new FakeAdmin().schedule)

        expect(rows.map(row => row.set)).toEqual(["default", "calm", "frozen", "rush", "default"])
        expect(rows.map(row => row.priority)).toEqual([0, 1, 2, 0, 0])
    })

    it("mark the entry playing, or the phase that runs when none plays", () => {
        const schedule = new FakeAdmin().schedule
        const rows = rowsOf(schedule)
        expect(playingOf(schedule, rows, new Date())).toBe("entry-running")

        const phaseOnly = new GetScheduleResponse({...schedule, playing: new Playing({ruleSet: "rush"})})
        expect(playingOf(phaseOnly, rows, new Date("2099-10-31T22:00:00Z"))).toBe(rows[3].key)
    })

    it("change only while an entry has not started", () => {
        const rows = rowsOf(new FakeAdmin().schedule)
        const now = new Date()

        expect(rows.map(row => changeable(row, now))).toEqual([false, false, true, false, false])
    })
})

describe("the sets", () => {
    it("an entry plays are the ones both modules keep", () => {
        const admin = new FakeAdmin()

        expect(playable(admin.rules, admin.antibotRules)).toEqual(["default"])
        expect(lacking("rush", admin.rules, admin.antibotRules)).toEqual(["planet", "the antibot"])
        expect(lacking("default", admin.rules, admin.antibotRules)).toEqual([])
    })
})

describe("an entry's draft", () => {
    it("starts empty on a set other than default, and is complete once each time is typed", () => {
        const draft = draftOf(undefined, ["default", "rush"])

        expect(draft).toEqual({set: "rush", startsAt: "", endsAt: "", priority: "1"})
        expect(complete(draft)).toBe(false)
        expect(complete({...draft, startsAt: "2099-10-31T20:00", endsAt: "2099-10-31T23:00"})).toBe(true)
        expect(complete({...draft, startsAt: "2099-10-31T20:00", endsAt: "2099-10-31T23:00", priority: "0"})).toBe(false)
    })

    it("of an entry kept is its values in the operator's time", () => {
        const at = new Date(2099, 5, 1, 20, 5)
        const entry = new FakeAdmin().schedule.entries[1]
        entry.startsAt = Timestamp.fromDate(at)

        expect(draftOf(entry, []).startsAt).toBe("2099-06-01T20:05")
    })
})
