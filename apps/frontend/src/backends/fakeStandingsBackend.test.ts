import {afterEach, describe, expect, it, vi} from "vitest"
import {FakePlayer, FakeStandingsBackend, MOVE_EVERY_MS} from "./fakeStandingsBackend.ts"
import {NameColor} from "./player.ts"
import {Race, Standing} from "./standings.ts"

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

        const stop = backend.listenForStandings("fr", (standings) => latest = standings)
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

        backend.listenForStandings("", () => {})()
        await vi.advanceTimersByTimeAsync(MOVE_EVERY_MS * 3)

        expect(random).not.toHaveBeenCalled()
    })
})

describe("FakeStandingsBackend.listenForRace", () => {
    it("ranks the flags by the tiles the players hold today, in a day that ends at 21:00 UTC", async () => {
        vi.useFakeTimers()
        const now = Date.UTC(2026, 9, 16, 22)
        const backend = new FakeStandingsBackend(PLAYERS, () => 0, () => now)
        let latest: Race | undefined

        const stop = backend.listenForRace((race) => latest = race)
        await vi.advanceTimersByTimeAsync(0)

        expect(latest?.round?.endsAt).toBe(Date.UTC(2026, 9, 17, 21))
        expect(latest?.round?.standings.map((s) => `${s.rank} ${s.countryCode} ${s.points}`)).toEqual(["1 fr 25", "2 gh 18"])
        expect(latest?.scores[0]).toEqual({rank: 1, countryCode: "fr", points: 61, roundsWon: 2})
        expect(latest?.closed).toBeUndefined()
        stop()
    })

    it("closes the day on demand: its results, and the season's table before and after them", async () => {
        vi.useFakeTimers()
        const now = Date.UTC(2026, 9, 16, 21)
        const backend = new FakeStandingsBackend(PLAYERS, () => 0, () => now)
        let latest: Race | undefined
        const stop = backend.listenForRace((race) => latest = race)
        await vi.advanceTimersByTimeAsync(0)

        const closed = backend.closeRound()

        expect(closed.number).toBe(5)
        expect(closed.endedAt).toBe(now)
        expect(closed.standings.map((s) => `${s.rank} ${s.countryCode} ${s.points}`)).toEqual(["1 fr 25", "2 gh 18"])
        expect(closed.before[0]).toEqual({rank: 1, countryCode: "fr", points: 61, roundsWon: 2})
        expect(closed.after[0]).toEqual({rank: 1, countryCode: "fr", points: 86, roundsWon: 3})
        expect(closed.after.find((score) => score.countryCode === "gh")).toMatchObject({points: 18})
        expect(latest?.closed).toEqual(closed)
        expect(latest?.round?.number).toBe(6)
        stop()
    })

    it("shows a battle's round of its own while it is open, at its times", async () => {
        vi.useFakeTimers()
        const now = Date.UTC(2026, 9, 31, 21)
        const backend = new FakeStandingsBackend(PLAYERS, () => 0, () => now)
        let latest: Race | undefined
        const stop = backend.listenForRace((race) => latest = race)
        await vi.advanceTimersByTimeAsync(0)

        backend.openBattle(now, now + 7_200_000, 3)

        expect(latest?.round).toMatchObject({number: 1, endsAt: now + 7_200_000, own: true, times: 3})
        expect(latest?.round?.standings.map((s) => s.points)).toEqual([75, 54])
        stop()
    })

    it("closes a battle with its numbers: triple points, the tiles each flag took and its top players", () => {
        const now = Date.UTC(2026, 9, 31, 23)
        const backend = new FakeStandingsBackend(PLAYERS, () => 0.5, () => now)
        backend.openBattle(now - 7_200_000, now, 3)

        const closed = backend.closeBattle(true)

        expect(closed).toMatchObject({number: 1, endedAt: now, startedAt: now - 7_200_000, finale: true, own: true, times: 3})
        expect(closed.standings.map((s) => s.points)).toEqual([75, 54])
        expect(closed.takes).toEqual([{countryCode: "fr", tiles: 5}, {countryCode: "gh", tiles: 2}])
        expect(closed.players.map((p) => `${p.rank} ${p.name} ${p.countryCode} ${p.tiles}`)).toEqual(["1 Ana fr 3", "2 Kofi gh 2", "3 Bastien fr 1"])
    })
})
