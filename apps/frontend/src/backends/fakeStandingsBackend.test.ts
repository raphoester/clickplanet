import {afterEach, describe, expect, it, vi} from "vitest"
import {FakePlayer, FakeStandingsBackend, MOVE_EVERY_MS} from "./fakeStandingsBackend.ts"
import {NameColor} from "./player.ts"
import {Round, Score, Standing} from "./standings.ts"

const PLAYERS: FakePlayer[] = [
    {name: "Ana", color: NameColor.PINK, tiles: {fr: 30}},
    {name: "Kofi", color: NameColor.GREEN, tiles: {gh: 20, fr: 5}},
    {name: "Bastien", color: NameColor.YELLOW, tiles: {fr: 10}},
]

const namesOf = (standings: Standing[] | undefined) => standings?.map((standing) => `${standing.rank} ${standing.name} ${standing.tiles}`)

afterEach(() => vi.useRealTimers())

describe("FakeStandingsBackend.listenForStandings", () => {
    it("sends the view's top at once, then a new one each time another player takes tiles", async () => {
        vi.useFakeTimers()
        const draws = [0.99, 0, 0.99]
        const backend = new FakeStandingsBackend(PLAYERS, () => draws.shift() ?? 0)
        let latest: Standing[] | undefined

        const stop = backend.listenForStandings("season-0", "fr", (standings) => latest = standings)
        await vi.advanceTimersByTimeAsync(0)
        expect(namesOf(latest)).toEqual(["1 Ana 30", "2 Bastien 10", "3 Kofi 5"])

        await vi.advanceTimersByTimeAsync(MOVE_EVERY_MS)
        expect(namesOf(latest)).toEqual(["1 Bastien 50", "2 Ana 30", "3 Kofi 5"])
        stop()
    })

    it("moves nobody once nobody follows", async () => {
        vi.useFakeTimers()
        const random = vi.fn(() => 0)
        const backend = new FakeStandingsBackend(PLAYERS, random)

        backend.listenForStandings("season-0", "", () => {})()
        await vi.advanceTimersByTimeAsync(MOVE_EVERY_MS * 3)

        expect(random).not.toHaveBeenCalled()
    })
})

function following(backend: FakeStandingsBackend, roundIds: string[]) {
    const read: {scores?: Score[], rounds: Map<string, Round>} = {rounds: new Map()}
    const stop = backend.listenForRace("season-0", roundIds, {
        onScores: (scores) => read.scores = scores,
        onRound: (round) => read.rounds.set(round.roundId, round),
    })
    return {read, stop}
}

describe("FakeStandingsBackend.listenForRace", () => {
    it("ranks the flags by the tiles the players hold today, in each round asked for", async () => {
        vi.useFakeTimers()
        const backend = new FakeStandingsBackend(PLAYERS, () => 0, () => Date.UTC(2026, 9, 16, 22))
        const {read, stop} = following(backend, ["season-0/day-9"])
        await vi.advanceTimersByTimeAsync(0)

        const day = read.rounds.get("season-0/day-9")
        expect(day).toMatchObject({times: 1, endedAt: 0})
        expect(day?.standings.map((s) => `${s.rank} ${s.countryCode} ${s.points}`)).toEqual(["1 fr 25", "2 gh 18"])
        expect(read.scores?.[0]).toEqual({rank: 1, countryCode: "fr", points: 61, roundsWon: 2})
        stop()
    })

    it("closes the day on demand: its results, and the season's table before and after them", async () => {
        vi.useFakeTimers()
        const now = Date.UTC(2026, 9, 16, 21)
        const backend = new FakeStandingsBackend(PLAYERS, () => 0, () => now)
        const {read, stop} = following(backend, ["season-0/day-9"])
        await vi.advanceTimersByTimeAsync(0)

        const closed = backend.closeRound("season-0/day-9")

        expect(closed.roundId).toBe("season-0/day-9")
        expect(closed.endedAt).toBe(now)
        expect(closed.standings.map((s) => `${s.rank} ${s.countryCode} ${s.points}`)).toEqual(["1 fr 25", "2 gh 18"])
        expect(closed.before[0]).toEqual({rank: 1, countryCode: "fr", points: 61, roundsWon: 2})
        expect(closed.after[0]).toEqual({rank: 1, countryCode: "fr", points: 86, roundsWon: 3})
        expect(closed.after.find((score) => score.countryCode === "gh")).toMatchObject({points: 18})
        expect(read.rounds.get("season-0/day-9")).toEqual(closed)
        expect(await backend.round("season-0/day-9")).toEqual(closed)
        expect(read.scores?.[0]).toEqual(closed.after[0])
        stop()
    })

    it("shows a battle's round while it is open, at its times", async () => {
        vi.useFakeTimers()
        const now = Date.UTC(2026, 9, 31, 21)
        const backend = new FakeStandingsBackend(PLAYERS, () => 0, () => now)
        const {read, stop} = following(backend, ["entry-1"])
        await vi.advanceTimersByTimeAsync(0)

        backend.openBattle("entry-1", now, 3)

        expect(read.rounds.get("entry-1")).toMatchObject({startedAt: now, endedAt: 0, times: 3})
        expect(read.rounds.get("entry-1")?.standings.map((s) => s.points)).toEqual([75, 54])
        stop()
    })

    it("closes a battle with its numbers: triple points, the tiles each flag took and its top players", async () => {
        const now = Date.UTC(2026, 9, 31, 23)
        const backend = new FakeStandingsBackend(PLAYERS, () => 0.5, () => now)
        backend.openBattle("entry-final", now - 7_200_000, 3)

        const closed = backend.closeBattle()

        expect(closed).toMatchObject({roundId: "entry-final", endedAt: now, startedAt: now - 7_200_000, times: 3})
        expect(closed.standings.map((s) => s.points)).toEqual([75, 54])
        expect(closed.takes).toEqual([{countryCode: "fr", tiles: 5}, {countryCode: "gh", tiles: 2}])
        expect(closed.players.map((p) => `${p.rank} ${p.name} ${p.countryCode} ${p.tiles}`)).toEqual(["1 Ana fr 3", "2 Kofi gh 2", "3 Bastien fr 1"])
        expect(await backend.round("entry-final")).toEqual(closed)
        expect(await backend.round("entry-other")).toBeUndefined()
    })
})
