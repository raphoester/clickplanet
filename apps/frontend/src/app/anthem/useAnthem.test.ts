// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, renderHook} from "@testing-library/react"
import {HOLD_MS} from "../../domain/anthemLeader.ts"
import {Countries} from "../../domain/countries.ts"
import {LeaderboardEntry} from "../../domain/leaderboard.ts"
import {CountryOrder} from "../../domain/race.ts"
import {DEFAULT_SOUND_SETTINGS} from "../../domain/soundSettings.ts"
import {useAnthem} from "./useAnthem.ts"
import {useSoundSettings} from "../sound/soundSettingsStore.ts"

type Shown = {board: LeaderboardEntry[], order: CountryOrder}

const board = (...codes: string[]): LeaderboardEntry[] =>
    codes.map((code, index) => ({country: Countries.get(code)!, tiles: 100 - index}))
const follow = (shown: Shown) =>
    renderHook(({board, order}: Shown) => useAnthem(board, order, false), {initialProps: shown})

beforeEach(() => {
    useSoundSettings.setState({settings: DEFAULT_SOUND_SETTINGS})
    vi.useFakeTimers({now: 0})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("useAnthem", () => {
    it("plays nothing before there is a board", () => {
        const {result} = follow({board: [], order: "points"})
        expect(result.current.code).toBeUndefined()
    })

    it("plays the leader of the order picked, at once", () => {
        const {result, rerender} = follow({board: board("pl", "fr", "pt"), order: "points"})
        expect(result.current.code).toBe("pl")

        rerender({board: board("pt", "pl", "fr"), order: "tiles"})
        expect(result.current.code).toBe("pt")

        rerender({board: board("pl", "fr", "pt"), order: "points"})
        expect(result.current.code).toBe("pl")
    })

    it("still holds a new leader of the same order", () => {
        const {result, rerender} = follow({board: board("pt", "hu"), order: "tiles"})

        rerender({board: board("hu", "pt"), order: "tiles"})
        expect(result.current.code).toBe("pt")

        act(() => vi.advanceTimersByTime(HOLD_MS - 1000))
        expect(result.current.code).toBe("pt")

        act(() => vi.advanceTimersByTime(1000))
        expect(result.current.code).toBe("hu")
    })
})
