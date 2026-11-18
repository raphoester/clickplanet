// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import LevelUp from "./LevelUp.tsx"
import {useLevel} from "./levelStore.ts"

beforeEach(() => {
    vi.useFakeTimers({shouldAdvanceTime: true})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    useLevel.setState(useLevel.getInitialState(), true)
})

const LADDER = {spreadClicks: 2, shields: 3, encloseClicks: 5, bomb: 7, virus: 10, mine: 13, rally: 15}

const twelve = {level: 12, xp: 18_700, levelXp: 18_634, nextLevelXp: 24_192}

describe("LevelUp", () => {
    it("names the level reached and plays the fanfare", () => {
        const play = vi.fn()
        render(<LevelUp level={twelve} play={play} onClose={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "Level up! Level 12"})).toBeDefined()
        expect(play).toHaveBeenCalledWith("title")
    })

    it("closes only once the reveal has played", async () => {
        const onClose = vi.fn()
        render(<LevelUp level={twelve} play={vi.fn()} onClose={onClose}/>)
        expect(screen.queryByRole("button")).toBeNull()

        act(() => vi.advanceTimersByTime(TITLE_REVEAL.ready * 1000))
        await userEvent.setup({advanceTimers: vi.advanceTimersByTime}).click(screen.getByRole("button", {name: "OK"}))

        expect(onClose).toHaveBeenCalledTimes(1)
    })

    it("names the bonus the level unlocked, and the next", () => {
        useLevel.setState({unlocks: LADDER})
        render(<LevelUp level={{...twelve, level: 13}} from={12} play={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.getByText(/New bonus unlocked:/).textContent).toBe("New bonus unlocked: Mine")
        expect(screen.getByText(/Next:/).textContent).toBe("Next: Rally, unlocks at level 15")
    })

    it("names every bonus a jump passed", () => {
        useLevel.setState({unlocks: LADDER})
        render(<LevelUp level={{...twelve, level: 11}} from={6} play={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.getAllByText(/New bonus unlocked:/).map((line) => line.textContent))
            .toEqual(["New bonus unlocked: Bomb", "New bonus unlocked: Virus"])
    })

    it("names no unlock for a level that has none, nor a next one past the ladder", () => {
        useLevel.setState({unlocks: LADDER})
        render(<LevelUp level={{...twelve, level: 25}} from={24} play={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.queryByText(/New bonus unlocked:/)).toBeNull()
        expect(screen.queryByText(/Next:/)).toBeNull()
    })

    it("names no unlock for the account's first level, which it knew none before", () => {
        useLevel.setState({unlocks: LADDER})
        render(<LevelUp level={{...twelve, level: 9}} play={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.queryByText(/New bonus unlocked:/)).toBeNull()
        expect(screen.getByText(/Next:/).textContent).toBe("Next: Virus, unlocks at level 10")
    })
})
