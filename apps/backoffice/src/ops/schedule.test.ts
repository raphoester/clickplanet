import {Timestamp} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {EntryKind} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {changeable, complete, draftOf, ended, kinds, lacking, playable, questionOf, rowsOf, shortOf, wordOf} from "./schedule.ts"

describe("the schedule's rows", () => {
    it("come by start", () => {
        expect(rowsOf(new FakeAdmin().schedule).map(row => row.entry.ruleSet)).toEqual(["rush", "calm", "frozen"])
    })

    it("change only while an entry has not started, and end at their end", () => {
        const rows = rowsOf(new FakeAdmin().schedule)
        const now = new Date()

        expect(rows.map(row => changeable(row, now))).toEqual([false, false, true])
        expect(rows.map(row => ended(row, now))).toEqual([true, false, false])
    })
})

describe("the sets", () => {
    it("an entry plays are the ones every module keeps", () => {
        const admin = new FakeAdmin()
        const modules = {planet: admin.rules, antibot: admin.antibotRules, scoring: admin.seasonRules}

        expect(playable(modules)).toEqual(["default"])
        expect(lacking("rush", modules)).toEqual(["planet", "the antibot"])
        expect(lacking("frozen", modules)).toEqual(["planet", "the antibot", "scoring"])
        expect(lacking("default", modules)).toEqual([])
    })
})

describe("the kinds", () => {
    it("are every kind the proto names but unspecified, each with its word", () => {
        expect(kinds()).toEqual([EntryKind.PLAIN, EntryKind.BATTLE])
        expect(kinds().map(wordOf)).toEqual(["plain", "battle"])
    })
})

describe("an entry's draft", () => {
    it("starts empty on a set other than default, as a battle, and is complete once each time is typed", () => {
        const draft = draftOf(undefined, ["default", "rush"])

        expect(draft).toEqual({set: "rush", kind: EntryKind.BATTLE, description: "", startsAt: "", endsAt: "", priority: "1"})
        expect(complete(draft)).toBe(false)
        const typed = {...draft, startsAt: "2099-10-31T20:00", endsAt: "2099-10-31T23:00"}
        expect(complete(typed)).toBe(true)
        expect(complete({...typed, priority: "-1"})).toBe(false)
        expect(complete({...typed, kind: EntryKind.UNSPECIFIED})).toBe(false)
        expect(complete({...typed, description: "é".repeat(300)})).toBe(true)
        expect(complete({...typed, description: "é".repeat(301)})).toBe(false)
    })

    it("of an entry kept is its description, shown short in the list", () => {
        const entry = new FakeAdmin().schedule.entries[0]

        expect(draftOf(entry, []).description).toBe("Triple points. A bonus box every minute.")
        expect(shortOf(entry.description)).toBe("Triple points. A bonus box every minute.")
        expect(shortOf("x".repeat(80))).toBe(`${"x".repeat(59)}…`)
    })

    it("asks with the set and the kind", () => {
        const draft = {...draftOf(undefined, ["rush"]), kind: EntryKind.PLAIN, startsAt: "2099-10-31T20:00", endsAt: "2099-10-31T23:00"}

        expect(questionOf(draft)).toMatch(/^Play rush as a plain entry from /)
    })

    it("of an entry kept is its kind", () => {
        expect(draftOf(new FakeAdmin().schedule.entries[2], []).kind).toBe(EntryKind.BATTLE)
    })

    it("of an entry kept is its values in the operator's time", () => {
        const at = new Date(2099, 5, 1, 20, 5)
        const entry = new FakeAdmin().schedule.entries[2]
        entry.startsAt = Timestamp.fromDate(at)

        expect(draftOf(entry, []).startsAt).toBe("2099-06-01T20:05")
    })
})
