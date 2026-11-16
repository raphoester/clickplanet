// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import LevelLine from "./LevelLine.tsx"
import {useLevel} from "./levelStore.ts"

afterEach(() => {
    cleanup()
    useLevel.getState().forget()
})

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
})
