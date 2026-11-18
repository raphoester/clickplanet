// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import LevelLine from "./LevelLine.tsx"
import {useLevel} from "./levelStore.ts"

afterEach(() => {
    cleanup()
    useLevel.getState().forget()
    vi.useRealTimers()
})

const twelve = {level: 12, xp: 20_000, levelXp: 18_634, nextLevelXp: 24_192}

describe("LevelLine", () => {
    it("draws nothing while the page has no level, as a guest's", () => {
        const {container} = render(<LevelLine/>)

        expect(container.innerHTML).toBe("")
    })

    it("draws the level and the XP gained on its bar", () => {
        render(<LevelLine/>)
        act(() => useLevel.getState().reached({level: 12, xp: 20_000, levelXp: 18_634, nextLevelXp: 24_192}))

        expect(screen.getByRole("img", {name: "Level 12"}).textContent).toBe("Lv 12")
        expect(screen.getByText(`${(1366).toLocaleString()} / ${(5558).toLocaleString()} XP`)).toBeDefined()
        const bar = screen.getByRole("meter", {name: "XP to level 13"})
        expect(bar.getAttribute("aria-valuenow")).toBe("1366")
        expect(bar.getAttribute("aria-valuemax")).toBe("5558")
        expect(bar.getAttribute("aria-valuetext")).toBe(`${(1366).toLocaleString()} / ${(5558).toLocaleString()} XP`)
    })

    it("shows a gain in the XP's place, with why, then the XP again", () => {
        vi.useFakeTimers()
        render(<LevelLine/>)
        act(() => useLevel.getState().reached(twelve))

        act(() => useLevel.getState().gained({gains: [{source: "daysPlayed", xp: 100}], level: {...twelve, xp: 20_100}}))

        expect(screen.getByText("+100 XP")).toBeDefined()
        expect(screen.getByText("First take of the day")).toBeDefined()
        expect(screen.getByRole("status").textContent).toBe("+100 XP First take of the day")
        expect(screen.getByRole("meter").getAttribute("aria-valuenow")).toBe("1466")

        act(() => vi.runAllTimers())

        expect(screen.queryByText("+100 XP")).toBeNull()
        expect(screen.getByText(`${(1466).toLocaleString()} / ${(5558).toLocaleString()} XP`)).toBeDefined()
    })

    it("counts the takes up as the player clicks, and reads none of them out", () => {
        render(<LevelLine/>)
        act(() => useLevel.getState().reached(twelve))

        for (let i = 0; i < 3; i++) act(() => useLevel.getState().gained({gains: [{source: "tilesTaken", xp: 1}]}))

        expect(screen.getByText("+3 XP")).toBeDefined()
        expect(screen.getByText("Tiles taken")).toBeDefined()
        expect(screen.queryByRole("status")).toBeNull()
    })
})
