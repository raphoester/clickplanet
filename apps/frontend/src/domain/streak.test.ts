import {describe, expect, it} from "vitest"
import {streakShown} from "./streak.ts"

describe("streakShown", () => {
    it("shows a run of three days or more", () => {
        expect(streakShown(3)).toBe(true)
        expect(streakShown(365)).toBe(true)
    })

    it("does not show a run of two days or less", () => {
        expect(streakShown(2)).toBe(false)
        expect(streakShown(1)).toBe(false)
        expect(streakShown(0)).toBe(false)
    })
})
