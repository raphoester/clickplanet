// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, renderHook} from "@testing-library/react"
import {HOLD_MS} from "../../domain/anthemLeader.ts"
import {Countries} from "../../domain/countries.ts"
import {LeaderboardEntry} from "../../domain/leaderboard.ts"
import {CountryOrder} from "../../domain/race.ts"
import {DEFAULT_SOUND_SETTINGS} from "../../domain/soundSettings.ts"
import {FlagCatalog} from "../../domain/flags.ts"
import {useFlags} from "../flags/flagsStore.ts"
import {useAnthem} from "./useAnthem.ts"
import {useSoundSettings} from "../sound/soundSettingsStore.ts"

type Shown = {board: LeaderboardEntry[], order: CountryOrder}

const board = (...codes: string[]): LeaderboardEntry[] =>
    codes.map((code, index) => ({country: Countries.get(code)!, tiles: 100 - index}))
const follow = (shown: Shown) =>
    renderHook(({board, order}: Shown) => useAnthem(board, order), {initialProps: shown})

beforeEach(() => {
    useSoundSettings.setState({settings: DEFAULT_SOUND_SETTINGS})
    vi.useFakeTimers({now: 0})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    useFlags.setState({catalog: FlagCatalog.BUILT_IN})
})

const EMPIRE = "r-jz9cwqzp0k"
const sold = (anthem?: string) => FlagCatalog.BUILT_IN.withFlags([{
    code: EMPIRE, name: "Heiliges Römisches Reich", closed: false,
    sold: {picture: {url: "x.png", width: 768, height: 512, stretch: false, focusX: 0.5, focusY: 0.5}, endsAt: 0, ...anthem ? {anthem} : {}},
}])

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

    it("plays the recording a sold flag's holder chose, and none for none", () => {
        useFlags.setState({catalog: sold("de")})
        const leading = [{country: {code: EMPIRE}, tiles: 100}]
        const {result} = follow({board: leading, order: "tiles"})
        expect(result.current.code).toBe(EMPIRE)
        expect(result.current.title).toBe("Das Lied der Deutschen")

        act(() => useFlags.setState({catalog: sold()}))
        expect(result.current.title).toBeUndefined()
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
