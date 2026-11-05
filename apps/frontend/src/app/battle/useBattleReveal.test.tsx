// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {ClosedRound} from "../../backends/standings.ts"
import {Battle, BATTLE_SEEN_STORAGE_KEY} from "../../domain/battle.ts"
import {DAY_BRIEF_WAIT_MS} from "../../domain/battleBrief.ts"
import {BattleRevealState, useBattleReveal} from "./useBattleReveal.ts"

const battle: Battle = {entryId: "entry-1", since: 1, endsAt: 2, description: "", roundId: "entry-1"}
const later: Battle = {entryId: "entry-2", since: 3, endsAt: 4, description: "", roundId: "entry-2"}

const dayEnded = (endedAt: number): ClosedRound => ({
    moment: "day", season: 0, number: 24, roundId: "season-0/day-24", endedAt, finale: false, standings: [], before: [],
    after: [], times: 1, startedAt: 0, entryId: "", takes: [], players: [],
})

const read = dayEnded(3)

let latest: BattleRevealState

function Harness({battle, closed}: {battle?: Battle, closed: ClosedRound | undefined}) {
    latest = useBattleReveal(battle, closed)
    return null
}

beforeEach(() => {
    window.localStorage.clear()
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("useBattleReveal", () => {
    it("announces a battle once per browser", () => {
        const {rerender} = render(<Harness battle={battle} closed={read}/>)
        expect(latest.battle).toEqual(battle)

        act(() => latest.dismiss())
        expect(latest.battle).toBeUndefined()
        expect(window.localStorage.getItem(BATTLE_SEEN_STORAGE_KEY)).toBe("entry-1")

        rerender(<Harness battle={later} closed={read}/>)
        expect(latest.battle).toEqual(later)
    })

    it("remembers across a reload", () => {
        window.localStorage.setItem(BATTLE_SEEN_STORAGE_KEY, "entry-1")
        render(<Harness battle={battle} closed={read}/>)
        expect(latest.battle).toBeUndefined()
    })

    it("announces nothing with no battle", () => {
        render(<Harness closed={read}/>)
        expect(latest.battle).toBeUndefined()
    })

    it("waits for the day the battle ended to be debriefed", () => {
        const {rerender} = render(<Harness battle={later} closed={dayEnded(1)}/>)
        expect(latest.battle).toBeUndefined()

        rerender(<Harness battle={later} closed={read}/>)
        expect(latest.battle).toEqual(later)
    })

    it("waits no longer than the limit when the day's debrief never comes", () => {
        vi.useFakeTimers()
        render(<Harness battle={later} closed={undefined}/>)
        expect(latest.battle).toBeUndefined()

        act(() => vi.advanceTimersByTime(DAY_BRIEF_WAIT_MS))
        expect(latest.battle).toEqual(later)
    })
})
