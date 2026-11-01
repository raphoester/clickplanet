// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {PLAIN_TURN, TurnBackend} from "../../backends/turn.ts"
import {FakeTurnBackend, SEASON_ZERO} from "../../backends/fakeTurnBackend.ts"
import {Schedule, useSchedule} from "./useSchedule.ts"

const HOUR = 3_600_000

let latest: Schedule | undefined

function Harness({backend}: {backend?: TurnBackend}) {
    latest = useSchedule(backend)
    return null
}

afterEach(cleanup)

describe("useSchedule", () => {
    it("knows no turn and no battle with no backend", () => {
        render(<Harness/>)
        expect(latest).toEqual({turn: undefined, battles: [], seasons: {ended: []}, debrief: undefined})
    })

    it("holds the turn, the battles, the seasons and the day the backend sends first", () => {
        render(<Harness backend={new FakeTurnBackend(SEASON_ZERO, () => Date.UTC(2026, 9, 28, 12))}/>)
        expect(latest).toEqual({
            turn: PLAIN_TURN,
            battles: [],
            seasons: {
                current: SEASON_ZERO,
                today: {number: 21, startsAt: Date.UTC(2026, 9, 27, 21), endsAt: Date.UTC(2026, 9, 28, 21), roundId: "season-0/day-21"},
                ended: [],
            },
            debrief: undefined,
        })
    })

    it("holds the last debrief, and the next day once a day ended", () => {
        const backend = new FakeTurnBackend(SEASON_ZERO, () => Date.UTC(2026, 9, 28, 12))
        render(<Harness backend={backend}/>)

        act(() => {
            backend.endDay()
        })

        expect(latest?.debrief).toEqual({kind: "day", season: 0, number: 21, roundId: "season-0/day-21", endedAt: Date.UTC(2026, 9, 28, 12)})
        expect(latest?.seasons.today).toMatchObject({number: 22, roundId: "season-0/day-22"})
    })

    it("follows each change of the turn and of the battles to come", () => {
        const backend = new FakeTurnBackend()
        render(<Harness backend={backend}/>)

        let id = ""
        act(() => {
            id = backend.scheduleBattle(Date.now() + HOUR, 120)
        })
        expect(latest?.battles.map((battle) => battle.entryId)).toEqual([id])

        act(() => backend.moveBattle(id, Date.now() + 2 * HOUR))
        expect(latest?.battles[0].startsAt).toBeGreaterThan(Date.now() + HOUR)

        act(() => backend.deleteBattle(id))
        expect(latest?.battles).toEqual([])

        act(() => backend.startBattle("entry-final", Date.now(), Date.now() + HOUR, "Triple points."))
        expect(latest?.turn).toMatchObject({kind: "battle", description: "Triple points.", entryId: "entry-final"})
    })

    it("stops following once unmounted", () => {
        const backend = new FakeTurnBackend()
        const {unmount} = render(<Harness backend={backend}/>)
        unmount()

        act(() => backend.startBattle("entry-final", Date.now(), Date.now() + HOUR))
        expect(latest?.turn).toEqual(PLAIN_TURN)
    })
})
