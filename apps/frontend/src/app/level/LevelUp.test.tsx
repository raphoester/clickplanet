// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import LevelUp from "./LevelUp.tsx"

beforeEach(() => {
    vi.useFakeTimers({shouldAdvanceTime: true})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

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
})
