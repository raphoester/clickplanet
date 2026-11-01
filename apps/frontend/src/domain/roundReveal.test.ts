import {describe, expect, it} from "vitest"
import {NameColor} from "../backends/player.ts"
import {ClosedRound, RoundStanding, Score} from "../backends/standings.ts"
import {Battle} from "./battle.ts"
import {
    battleFinishedLine,
    medalOf,
    moveOf,
    podiumOf,
    REVEAL_WITHIN_MS,
    revealFollowed,
    revealKeyOf,
    revealRowsOf,
    revealsDue,
    roundOverLine,
    tablePodiumOf,
} from "./roundReveal.ts"

const result = (rank: number, countryCode: string, points: number): RoundStanding => ({rank, countryCode, share: 0.1, points})
const score = (rank: number, countryCode: string, points: number): Score => ({rank, countryCode, points, roundsWon: 0})

const ENDED_AT = Date.UTC(2026, 9, 16, 21)
const DAY: ClosedRound = {
    moment: "day",
    season: 0,
    number: 5,
    roundId: "season-0/day-5",
    endedAt: ENDED_AT,
    finale: false,
    standings: [result(1, "fr", 25), result(2, "de", 18), result(3, "es", 15), result(4, "it", 12)],
    before: [score(1, "de", 50), score(2, "br", 40), score(3, "fr", 30)],
    after: [score(1, "de", 68), score(2, "fr", 55), score(3, "br", 40), score(4, "es", 15), score(5, "it", 12)],
    times: 1,
    startedAt: 0,
    entryId: "",
    takes: [],
    players: [],
}
const BATTLE: ClosedRound = {
    ...DAY, moment: "battle", number: 1, roundId: "entry-final", times: 3, startedAt: ENDED_AT - 7_200_000, entryId: "entry-final",
    players: [{rank: 1, accountId: "account-ana", name: "Ana", color: NameColor.UNSPECIFIED, countryCode: "fr", tiles: 9}],
}
const FINAL_BATTLE: ClosedRound = {...BATTLE, finale: true}
const SEASON: ClosedRound = {...FINAL_BATTLE, moment: "season"}

const kinds = (closed: ClosedRound | undefined, seen: string | null, battle?: Battle) =>
    revealsDue(closed, ENDED_AT + 60_000, seen, battle).map((reveal) => reveal.kind)
const seenBattle = (entryId: string): Battle => ({entryId, since: BATTLE.startedAt, endsAt: ENDED_AT, description: "", roundId: entryId})

describe("revealsDue", () => {
    it("plays a day closed in the last hours that this browser has not seen", () => {
        expect(kinds(DAY, null)).toEqual(["round"])
        expect(revealsDue(DAY, ENDED_AT + REVEAL_WITHIN_MS - 1, "0:4:1")).toEqual([{kind: "round", closed: DAY}])
    })

    it("plays a battle's debrief, three screens, and the season's results after the round that ended the season", () => {
        expect(kinds(BATTLE, null)).toEqual(["over", "points", "players"])
        expect(kinds(FINAL_BATTLE, null)).toEqual(["over", "points", "players", "results"])
    })

    it("skips a debrief screen with nothing in it", () => {
        expect(kinds({...BATTLE, standings: [], players: []}, null)).toEqual(["over"])
    })

    it("plays the debrief of the battle the page saw, and none of another battle's", () => {
        expect(kinds(BATTLE, null, seenBattle("entry-final"))).toEqual(["over", "points", "players"])
        expect(kinds(BATTLE, null, seenBattle("entry-before"))).toEqual([])
        expect(kinds(FINAL_BATTLE, null, seenBattle("entry-before"))).toEqual(["results"])
    })

    it("does not play a reveal twice, and keeps what comes after the one closed", () => {
        expect(kinds(DAY, revealKeyOf({kind: "round", closed: DAY}))).toEqual([])
        expect(kinds(FINAL_BATTLE, revealKeyOf({kind: "points", closed: FINAL_BATTLE}))).toEqual(["players", "results"])
        expect(kinds(FINAL_BATTLE, revealKeyOf({kind: "results", closed: FINAL_BATTLE}))).toEqual([])
    })

    it("tells a day from the battle of the same number and end", () => {
        expect(kinds(BATTLE, revealKeyOf({kind: "round", closed: {...BATTLE, moment: "day"}}))).toEqual(["over", "points", "players"])
    })

    it("plays the season's results, after its Final Battle's debrief when a battle finished it", () => {
        expect(kinds(SEASON, null)).toEqual(["over", "points", "players", "results"])
        expect(kinds(SEASON, null, seenBattle("entry-before"))).toEqual(["over", "points", "players", "results"])
        expect(kinds({...SEASON, entryId: "", number: 2, roundId: "season-0/finale"}, null)).toEqual(["over", "points", "players", "results"])
        expect(kinds({...SEASON, entryId: "", number: 0}, null)).toEqual(["results"])
    })

    it("does not play the Final Battle's screens again when the season's debrief follows its own", () => {
        expect(kinds(SEASON, revealKeyOf({kind: "players", closed: FINAL_BATTLE}))).toEqual(["results"])
        expect(kinds(SEASON, revealKeyOf({kind: "results", closed: FINAL_BATTLE}))).toEqual([])
    })

    it("does not play a round closed long ago, or no round at all", () => {
        expect(revealsDue(DAY, ENDED_AT + REVEAL_WITHIN_MS, null)).toEqual([])
        expect(revealsDue(undefined, ENDED_AT, null)).toEqual([])
    })
})

describe("podiumOf", () => {
    it("is the three countries that placed best", () => {
        expect(podiumOf(DAY).map((standing) => standing.countryCode)).toEqual(["fr", "de", "es"])
    })

    it("keeps a tie on the podium and never shows more than three", () => {
        const tied = {...DAY, standings: [result(1, "fr", 25), result(1, "de", 25), result(3, "es", 15), result(3, "it", 15)]}

        expect(podiumOf(tied).map((standing) => `${standing.rank} ${standing.countryCode}`)).toEqual(["1 fr", "1 de", "3 es"])
    })

    it("is empty when nobody scored", () => {
        expect(podiumOf({...DAY, standings: []})).toEqual([])
    })
})

describe("tablePodiumOf", () => {
    it("is the three countries first in a book's table, a tie kept and never more than three", () => {
        expect(tablePodiumOf(DAY.after).map((place) => place.countryCode)).toEqual(["de", "fr", "br"])
        expect(tablePodiumOf([score(1, "de", 90), score(1, "fr", 90), score(3, "br", 40), score(3, "es", 40)])
            .map((place) => `${place.rank} ${place.countryCode}`)).toEqual(["1 de", "1 fr", "3 br"])
    })

    it("is empty in a book nobody scored in", () => {
        expect(tablePodiumOf([])).toEqual([])
    })
})

describe("revealRowsOf", () => {
    const rows = revealRowsOf(DAY, "fr")

    it("ends in the season's order after the round, with the points each country gained", () => {
        expect(rows.map((row) => `${row.to} ${row.countryCode} ${row.after.points} +${row.gained}`)).toEqual([
            "0 de 68 +18",
            "1 fr 55 +25",
            "2 br 40 +0",
            "3 es 15 +15",
            "4 it 12 +12",
        ])
    })

    it("starts in the season's order before the round, with the countries new to the table below", () => {
        expect([...rows].sort((a, b) => a.from - b.from).map((row) => row.countryCode)).toEqual(["de", "br", "fr", "es", "it"])
    })

    it("marks the player's country", () => {
        expect(rows.filter((row) => row.mine).map((row) => row.countryCode)).toEqual(["fr"])
    })

    it("cuts the table and adds the player's country under it", () => {
        expect(revealRowsOf(DAY, "it", 2).map((row) => `${row.to} ${row.countryCode}`)).toEqual(["0 de", "1 fr", "2 it"])
    })

    it("says how each country moved", () => {
        expect(rows.map(moveOf)).toEqual([
            {kind: "same"},
            {kind: "up", by: 1},
            {kind: "down", by: 1},
            {kind: "new"},
            {kind: "new"},
        ])
    })
})

describe("roundOverLine", () => {
    it("names the day, the battle, or the Final Battle", () => {
        expect(roundOverLine({number: 5, finale: false, own: false})).toBe("Day 5 is over!")
        expect(roundOverLine({number: 23, finale: true, own: false})).toBe("Day 23 is over!")
        expect(roundOverLine({number: 1, finale: false, own: true})).toBe("The battle is over!")
        expect(roundOverLine({number: 2, finale: true, own: true})).toBe("The Final Battle is over!")
    })
})

describe("revealFollowed", () => {
    it("is true until the last screen", () => {
        expect(revealFollowed({kind: "over", closed: BATTLE})).toBe(true)
        expect(revealFollowed({kind: "players", closed: BATTLE})).toBe(false)
        expect(revealFollowed({kind: "players", closed: FINAL_BATTLE})).toBe(true)
        expect(revealFollowed({kind: "results", closed: FINAL_BATTLE})).toBe(false)
    })
})

describe("battleFinishedLine", () => {
    it("names the battle, or the Final Battle", () => {
        expect(battleFinishedLine({finale: false})).toBe("Battle finished!")
        expect(battleFinishedLine({finale: true})).toBe("Final Battle finished!")
    })
})

describe("medalOf", () => {
    it("is a medal for the podium and nothing below it", () => {
        expect([1, 2, 3, 4].map(medalOf)).toEqual(["🥇", "🥈", "🥉", ""])
    })
})
