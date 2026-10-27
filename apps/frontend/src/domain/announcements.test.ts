import {describe, expect, it} from "vitest"
import {PlayerTitle} from "../backends/player.ts"
import {ClosedRound} from "../backends/standings.ts"
import {Announcement, announced, announcementKeyOf} from "./announcements.ts"

const title = {id: "warlord", name: "Warlord"} as PlayerTitle
const battle = {since: 1, endsAt: 2, description: ""}
const closed = {season: 0, number: 5, endedAt: 3} as ClosedRound

const unlock: Announcement = {kind: "title", title}
const start: Announcement = {kind: "battle", battle}
const round: Announcement = {kind: "round", closed}

describe("the announcements", () => {
    it("are keyed by what they announce", () => {
        expect(announcementKeyOf(unlock)).toBe("title:warlord")
        expect(announcementKeyOf(start)).toBe("battle:1:2")
        expect(announcementKeyOf(round)).toBe("round:0:5:3")
    })

    it("queue in the order they arrive", () => {
        const queue = announced(announced(announced([], round), unlock), start)
        expect(queue).toEqual([round, unlock, start])
    })

    it("are announced once", () => {
        const queue = announced([unlock, round], {kind: "round", closed: {...closed}})
        expect(queue).toEqual([unlock, round])
    })
})
