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
        stop()
    })
})
