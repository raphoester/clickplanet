import {describe, expect, it} from "vitest"
import {DISCORD_BAR_RETURNS_AFTER_MS, isStillHidden} from "./discordBar.ts"

const pressedAt = 1_700_000_000_000

describe("isStillHidden", () => {
    it("holds from the press of the cross until the bar is due back", () => {
        const hiddenAt = String(pressedAt)

        expect(isStillHidden(hiddenAt, pressedAt)).toBe(true)
        expect(isStillHidden(hiddenAt, pressedAt + DISCORD_BAR_RETURNS_AFTER_MS - 1)).toBe(true)
        expect(isStillHidden(hiddenAt, pressedAt + DISCORD_BAR_RETURNS_AFTER_MS)).toBe(false)
    })

    it("brings the bar back the next day at the same hour", () => {
        expect(isStillHidden(String(pressedAt), pressedAt + 24 * 3600_000 - 60_000)).toBe(false)
    })

    it("hides nothing without a press it can read", () => {
        expect(isStillHidden(null, pressedAt)).toBe(false)
        expect(isStillHidden("", pressedAt)).toBe(false)
        expect(isStillHidden("hidden", pressedAt)).toBe(false)
    })

    it("hides nothing for a press in the future, after a clock was set back", () => {
        expect(isStillHidden(String(pressedAt + 1), pressedAt)).toBe(false)
    })
})
