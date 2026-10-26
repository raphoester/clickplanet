import {describe, expect, it} from "vitest"
import {magnifiedMost, SETTLE_SECONDS, SlotCandidate, SlotChoice} from "./flagSlot.ts"

const flag = (landmass: number, screenWidth: number, spriteWidth = 100, inView = true): SlotCandidate =>
    ({landmass, screenWidth, spriteWidth, inView})

describe("magnifiedMost", () => {
    it("is the widest flag on screen among those drawn wider than their sprite", () => {
        expect(magnifiedMost([flag(1, 300), flag(2, 900), flag(3, 600)])).toBe(2)
    })

    it("is nobody while every flag fits its sprite", () => {
        expect(magnifiedMost([flag(1, 100), flag(2, 149)])).toBeUndefined()
        expect(magnifiedMost([flag(1, 150)])).toBe(1)
        expect(magnifiedMost([])).toBeUndefined()
    })

    it("ignores a flag out of view, however wide", () => {
        expect(magnifiedMost([flag(1, 3000, 100, false), flag(2, 200)])).toBe(2)
        expect(magnifiedMost([flag(1, 3000, 100, false)])).toBeUndefined()
    })

    it("measures each flag against its own sprite", () => {
        expect(magnifiedMost([flag(1, 290, 200), flag(2, 160, 100)])).toBe(2)
        expect(magnifiedMost([flag(1, 320, 200), flag(2, 160, 100)])).toBe(1)
    })
})

describe("SlotChoice", () => {
    it("keeps its choice while the camera turns", () => {
        const choice = new SlotChoice()
        expect(choice.choose(false, 0, () => 1)).toBe(1)

        expect(choice.choose(true, 1, () => 2)).toBe(1)
        expect(choice.choose(true, 1.1, () => 2)).toBe(1)
        expect(choice.choose(false, 1.1 + SETTLE_SECONDS / 2, () => 2)).toBe(1)
    })

    it("changes its mind once the camera has been still for the settle time", () => {
        const choice = new SlotChoice()
        choice.choose(false, 0, () => 1)
        choice.choose(true, 1, () => 2)

        expect(choice.choose(false, 1 + SETTLE_SECONDS * 0.99, () => 2)).toBe(1)
        expect(choice.choose(false, 1 + SETTLE_SECONDS * 1.01, () => 2)).toBe(2)
    })

    it("follows a new best at once while the camera stays still", () => {
        const choice = new SlotChoice()
        choice.choose(false, 0, () => 1)

        expect(choice.choose(false, 0.01, () => 3)).toBe(3)
        expect(choice.choose(false, 0.02, () => undefined)).toBeUndefined()
    })

    it("does not ask for the best while it is holding", () => {
        const choice = new SlotChoice()
        let asked = 0
        choice.choose(true, 0, () => {
            asked++
            return 1
        })
        expect(asked).toBe(0)
    })
})
