import {describe, expect, it} from "vitest"
import {PlayerTitle} from "../backends/player.ts"
import {ClosedRound} from "../backends/standings.ts"
import {Announcement, announced, announcementKeyOf} from "./announcements.ts"

const title = {id: "warlord", name: "Warlord"} as PlayerTitle
const battle = {entryId: "entry-1", since: 1, endsAt: 2, description: "", roundId: "entry-1"}
const closed = {moment: "day", season: 0, number: 5, endedAt: 3} as ClosedRound

const unlock: Announcement = {kind: "title", title}
const start: Announcement = {kind: "battle", battle}
const clip: Announcement = {kind: "clip", appearance: {postId: "post-1", title: "FRANCE VS TURKEY", scheduledAt: 4, links: []}}
const round: Announcement = {kind: "round", closed}
const over: Announcement = {kind: "over", closed: {...closed, number: 1, moment: "battle", roundId: "entry-1"}}
const results: Announcement = {kind: "results", closed: {...closed, number: 1, moment: "battle", roundId: "entry-1"}}

describe("the announcements", () => {
    it("are keyed by what they announce", () => {
        expect(announcementKeyOf(unlock)).toBe("title:warlord")
        expect(announcementKeyOf(start)).toBe("battle:entry-1")
        expect(announcementKeyOf(clip)).toBe("clip:post-1")
        expect(announcementKeyOf(round)).toBe("round:0:5:3")
        expect(announcementKeyOf(over)).toBe("over:0:battle:entry-1:over")
        expect(announcementKeyOf(results)).toBe("results:0:results:entry-1")
    })

    it("queue in the order they arrive", () => {
        const queue = announced(announced(announced([], round), unlock), start)
        expect(queue).toEqual([round, unlock, start])
    })

    it("queue a battle's debrief before the season's results, whatever arrives between", () => {
        const queue = announced(announced(announced([], over), unlock), results)
        expect(queue).toEqual([over, unlock, results])
    })

    it("are announced once", () => {
        const queue = announced([unlock, round], {kind: "round", closed: {...closed}})
        expect(queue).toEqual([unlock, round])
    })
})
