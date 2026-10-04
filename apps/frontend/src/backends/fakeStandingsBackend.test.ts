import {afterEach, describe, expect, it, vi} from "vitest"
import {FakePlayer, FakeStandingsBackend, MOVE_EVERY_MS} from "./fakeStandingsBackend.ts"
import {NameColor} from "./player.ts"
import {Standing} from "./standings.ts"

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
