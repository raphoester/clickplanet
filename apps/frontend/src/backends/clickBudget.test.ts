import {describe, expect, it} from 'vitest'
import {ClickBudget, nextClickProgress, secondsToNextClick, tokensAt} from './clickBudget.ts'

function reading(overrides: Partial<ClickBudget> = {}): ClickBudget {
    return {tokens: 4, capacity: 10, perSecond: 1, readAt: 1_000, ...overrides}
}

describe("tokensAt", () => {
    it("reports the server's own reading at the moment it was taken", () => {
        expect(tokensAt(reading(), 1_000)).toBe(4)
    })

    it("replays the refill between two readings", () => {
        expect(tokensAt(reading(), 3_500)).toBe(6.5)
    })

    it("follows the server's rate rather than a rate of its own", () => {
        expect(tokensAt(reading({perSecond: 4}), 2_000)).toBe(8)
    })

    it("stops banking at the burst, as the server does", () => {
        expect(tokensAt(reading(), 1_000_000)).toBe(10)
    })

    it("never goes below zero, whatever is in flight", () => {
        expect(tokensAt(reading({tokens: -3}), 1_000)).toBe(0)
    })

    it("does not run backwards if the reading is somehow ahead of the clock", () => {
        expect(tokensAt(reading(), 0)).toBe(4)
    })
})

describe("nextClickProgress", () => {
    it("is how much of the next click has arrived", () => {
        expect(nextClickProgress(reading({tokens: 0}), 1_400)).toBeCloseTo(0.4)
    })

    it("ignores the clicks already in hand", () => {
        expect(nextClickProgress(reading({tokens: 6.25}), 1_000)).toBeCloseTo(0.25)
    })

    it("is whole at a full bucket, which is granting nothing back", () => {
        expect(nextClickProgress(reading({tokens: 10}), 1_000)).toBe(1)
    })
})

describe("secondsToNextClick", () => {
    it("is nothing while a click is in hand", () => {
        expect(secondsToNextClick(reading(), 1_000)).toBe(0)
    })

    it("is what is left of the token being granted back", () => {
        expect(secondsToNextClick(reading({tokens: 0.25}), 1_000)).toBeCloseTo(0.75)
    })

    it("scales with the server's rate", () => {
        expect(secondsToNextClick(reading({tokens: 0, perSecond: 2}), 1_000)).toBeCloseTo(0.5)
    })

    it("is forever when nothing is granted back", () => {
        expect(secondsToNextClick(reading({tokens: 0, perSecond: 0}), 1_000)).toBe(Infinity)
    })
})
