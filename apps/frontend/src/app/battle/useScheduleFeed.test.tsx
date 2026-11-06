// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, waitFor} from "@testing-library/react"
import {Season} from "../../backends/season.ts"
import {StandingsBackend} from "../../backends/standings.ts"
import {PLAIN_TURN, TurnBackend} from "../../backends/turn.ts"
import {FakeStandingsBackend} from "../../backends/fakeStandingsBackend.ts"
import {FakeTurnBackend, SEASON_ZERO} from "../../backends/fakeTurnBackend.ts"
import {NO_SCHEDULE, runningSeason, useSchedule} from "./scheduleStore.ts"
import {useScheduleFeed} from "./useScheduleFeed.ts"

const HOUR = 3_600_000

function Harness({turn, standings = new FakeStandingsBackend()}: {turn: TurnBackend, standings?: StandingsBackend}) {
    useScheduleFeed(turn, standings)
    return null
}

const schedule = () => useSchedule.getState()

afterEach(() => {
    cleanup()
    useSchedule.setState(NO_SCHEDULE)
    vi.useRealTimers()
})

describe("useScheduleFeed", () => {
    it("holds the turn, the battles, the seasons and the day the backend sends first", () => {
        render(<Harness turn={new FakeTurnBackend(SEASON_ZERO, () => Date.UTC(2026, 9, 28, 12))}/>)
        expect(schedule()).toMatchObject({
            turn: PLAIN_TURN,
            battle: undefined,
            battles: [],
            seasons: {
                current: SEASON_ZERO,
                today: {number: 21, startsAt: Date.UTC(2026, 9, 27, 21), endsAt: Date.UTC(2026, 9, 28, 21), roundId: "season-0/day-21"},
                ended: [],
            },
            debrief: undefined,
        })
    })

    it("holds the last debrief and its numbers once read, and the next day once a day ended", async () => {
        const backend = new FakeTurnBackend(SEASON_ZERO, () => Date.UTC(2026, 9, 28, 12))
        render(<Harness turn={backend}/>)

        act(() => {
            backend.endDay()
        })

        expect(schedule().debrief).toEqual({kind: "day", season: 0, number: 21, roundId: "season-0/day-21", endedAt: Date.UTC(2026, 9, 28, 12)})
        expect(schedule().seasons.today).toMatchObject({number: 22, roundId: "season-0/day-22"})
        await waitFor(() => expect(schedule().closed).toMatchObject({roundId: "season-0/day-21"}))
    })

    it("keeps the numbers read when a reconnect sends the same debrief again", async () => {
        const backend = new FakeTurnBackend(SEASON_ZERO, () => Date.UTC(2026, 9, 28, 12))
        render(<Harness turn={backend}/>)
        act(() => {
            backend.endDay()
        })
        await waitFor(() => expect(schedule().closed).toBeDefined())
        const closed = schedule().closed

        act(() => schedule().hearDebrief({...schedule().debrief!}))

        expect(schedule().closed).toBe(closed)
    })

    it("follows each change of the turn and of the battles to come", () => {
        const backend = new FakeTurnBackend()
        render(<Harness turn={backend}/>)

        let id = ""
        act(() => {
            id = backend.scheduleBattle(Date.now() + HOUR, 120)
        })
        expect(schedule().battles.map((battle) => battle.entryId)).toEqual([id])

        act(() => backend.moveBattle(id, Date.now() + 2 * HOUR))
        expect(schedule().battles[0].startsAt).toBeGreaterThan(Date.now() + HOUR)

        act(() => backend.deleteBattle(id))
        expect(schedule().battles).toEqual([])

        act(() => backend.startBattle("entry-final", Date.now(), Date.now() + HOUR, "Triple points."))
        expect(schedule().turn).toMatchObject({kind: "battle", description: "Triple points.", entryId: "entry-final"})
        expect(schedule().battle).toMatchObject({entryId: "entry-final", description: "Triple points."})
    })

    it("puts the store back and stops following once unmounted", () => {
        const backend = new FakeTurnBackend()
        const {unmount} = render(<Harness turn={backend}/>)
        unmount()

        act(() => backend.startBattle("entry-final", Date.now(), Date.now() + HOUR))
        expect(schedule().turn).toBeUndefined()
        expect(schedule().seasons.current).toBeUndefined()
    })

    it("has no season running while none runs", () => {
        render(<Harness turn={new FakeTurnBackend(undefined)}/>)
        expect(runningSeason(schedule())).toBeUndefined()
    })

    it("lets the season go at its end, even weeks away", () => {
        vi.useFakeTimers({now: Date.UTC(2026, 9, 3, 12)})
        const season: Season = {number: 0, startsAt: Date.UTC(2026, 9, 3), endsAt: Date.UTC(2026, 9, 31, 23), bookId: "season-0"}
        render(<Harness turn={new FakeTurnBackend(season)}/>)
        expect(runningSeason(schedule())).toEqual(season)

        act(() => {
            vi.advanceTimersByTime(season.endsAt - Date.now() - 1000)
        })
        expect(runningSeason(schedule())).toEqual(season)

        act(() => {
            vi.advanceTimersByTime(1000)
        })
        expect(runningSeason(schedule())).toBeUndefined()
    })

    it("runs the season again once its end moved", () => {
        vi.useFakeTimers({now: Date.UTC(2026, 9, 3, 12)})
        const season: Season = {number: 0, startsAt: Date.UTC(2026, 9, 3), endsAt: Date.UTC(2026, 9, 3, 13), bookId: "season-0"}
        render(<Harness turn={new FakeTurnBackend(season)}/>)
        act(() => {
            vi.advanceTimersByTime(HOUR)
        })
        expect(runningSeason(schedule())).toBeUndefined()

        const moved = {...season, endsAt: season.endsAt + HOUR}
        act(() => schedule().hearSeasons({...schedule().seasons, current: moved}))

        expect(runningSeason(schedule())).toEqual(moved)
    })
})
