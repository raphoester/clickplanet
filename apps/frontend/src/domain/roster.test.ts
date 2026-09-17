import {describe, expect, it} from "vitest"
import type {RosterEntry} from "../backends/player.ts"
import {rosterGroups} from "./roster.ts"

const entry = (name: string, guest: boolean): RosterEntry => ({name, tag: "4f2ca1", countryCode: "fr", guest, admin: false})

describe("rosterGroups", () => {
    it("puts players with a username in one group and guests in the other", () => {
        const groups = rosterGroups([entry("ana", false), entry("guest_Bo", true), entry("kiran_07", false)])

        expect(groups.players.map((e) => e.name)).toEqual(["ana", "kiran_07"])
        expect(groups.guests.map((e) => e.name)).toEqual(["guest_Bo"])
    })

    // The server sorted already; a second sort could only disagree with it.
    it("keeps the server's order inside each group", () => {
        const groups = rosterGroups([entry("guest_zed", true), entry("Bob", false), entry("guest_Al", true), entry("al", false)])

        expect(groups.players.map((e) => e.name)).toEqual(["Bob", "al"])
        expect(groups.guests.map((e) => e.name)).toEqual(["guest_zed", "guest_Al"])
    })

    it("answers two empty groups for an empty roster", () => {
        expect(rosterGroups([])).toEqual({players: [], guests: []})
    })
})
