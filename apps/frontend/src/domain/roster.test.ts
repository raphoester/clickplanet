import {describe, expect, it} from "vitest"
import type {RosterEntry} from "../backends/player.ts"
import {applyRosterEvent, rosterGroups} from "./roster.ts"

const entry = (name: string, guest: boolean, key = name): RosterEntry =>
    ({key, name, countryCode: "fr", guest, admin: false})

describe("rosterGroups", () => {
    it("puts players with a username in one group and guests in the other", () => {
        const groups = rosterGroups([entry("ana", false), entry("guest_Bo", true), entry("kiran_07", false)])

        expect(groups.players.map((e) => e.name)).toEqual(["ana", "kiran_07"])
        expect(groups.guests.map((e) => e.name)).toEqual(["guest_Bo"])
    })

    it("keeps the roster's order inside each group", () => {
        const groups = rosterGroups([entry("guest_zed", true), entry("Bob", false), entry("guest_Al", true), entry("al", false)])

        expect(groups.players.map((e) => e.name)).toEqual(["Bob", "al"])
        expect(groups.guests.map((e) => e.name)).toEqual(["guest_zed", "guest_Al"])
    })

    it("answers two empty groups for an empty roster", () => {
        expect(rosterGroups([])).toEqual({players: [], guests: []})
    })
})

describe("applyRosterEvent", () => {
    const ana = entry("ana", false)
    const bo = entry("guest_Bo", true)

    it("takes a whole roster as the server ordered it", () => {
        const roster = [bo, ana]

        expect(applyRosterEvent([ana], {kind: "roster", entries: roster})).toBe(roster)
    })

    it("puts a new line where the server would: players first, then by name ignoring case, then by key", () => {
        let roster = applyRosterEvent([], {kind: "entry", entry: bo})
        roster = applyRosterEvent(roster, {kind: "entry", entry: entry("Zed", false)})
        roster = applyRosterEvent(roster, {kind: "entry", entry: ana})
        roster = applyRosterEvent(roster, {kind: "entry", entry: entry("guest_Bo", true, "a")})

        expect(roster.map((e) => [e.name, e.key])).toEqual([
            ["ana", "ana"], ["Zed", "Zed"], ["guest_Bo", "a"], ["guest_Bo", "guest_Bo"],
        ])
    })

    it("replaces the line with the same key, and moves it when its name changed", () => {
        const guest = entry("guest_Yuki", true, "k1")

        const roster = applyRosterEvent([ana, bo, guest], {kind: "entry", entry: entry("Yuki", false, "k1")})

        expect(roster.map((e) => e.name)).toEqual(["ana", "Yuki", "guest_Bo"])
    })

    it("takes the line with the key out, and changes nothing for a key it does not hold", () => {
        const roster = [ana, bo]

        expect(applyRosterEvent(roster, {kind: "left", key: "ana"})).toEqual([bo])
        expect(applyRosterEvent(roster, {kind: "left", key: "nobody"})).toBe(roster)
    })
})
