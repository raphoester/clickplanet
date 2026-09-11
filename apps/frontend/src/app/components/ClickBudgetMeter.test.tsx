// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from 'vitest'
import {cleanup, render, screen} from '@testing-library/react'
import ClickBudgetMeter from './ClickBudgetMeter.tsx'
import {ClickBudget} from "../../backends/clickBudget.ts"

afterEach(cleanup)

function reading(overrides: Partial<ClickBudget> = {}): ClickBudget {
    return {tokens: 4, capacity: 10, perSecond: 1, readAt: performance.now(), ...overrides}
}

const meter = () => screen.getByRole("meter")
const pips = () => Array.from(document.querySelectorAll(".click-budget-pip"))
const tokens = () => meter().style.getPropertyValue("--click-budget-tokens")

describe("ClickBudgetMeter", () => {
    it("shows nothing against a server that reports no allowance", () => {
        const {container} = render(<ClickBudgetMeter/>)
        expect(container.innerHTML).toBe("")
    })

    it("shows the clicks in hand", () => {
        render(<ClickBudgetMeter budget={reading({tokens: 6.4})}/>)

        expect(screen.getByText("6")).toBeTruthy()
        expect(meter().getAttribute("aria-valuenow")).toBe("6")
    })

    it("draws one pip per click the server will bank", () => {
        render(<ClickBudgetMeter budget={reading({capacity: 5})}/>)

        expect(pips()).toHaveLength(5)
        expect(meter().getAttribute("aria-valuemax")).toBe("5")
    })

    // The pip count is the burst, so a backend that changes it changes this
    // with no frontend release — which is the point of sending the policy.
    it("follows the server's burst rather than a number of its own", () => {
        render(<ClickBudgetMeter budget={reading({capacity: 7})}/>)
        expect(pips()).toHaveLength(7)
    })

    it("becomes one bar past the count a row of pips can show", () => {
        render(<ClickBudgetMeter budget={reading({capacity: 40})}/>)

        expect(pips()).toHaveLength(0)
        expect(document.querySelector(".click-budget-bar")).toBeTruthy()
    })

    it("hands the fraction to the CSS, which is what fills the next pip", () => {
        render(<ClickBudgetMeter budget={reading({tokens: 3.5})}/>)
        expect(Number(tokens())).toBeCloseTo(3.5, 1)
    })

    it("each pip knows which click it stands for", () => {
        render(<ClickBudgetMeter budget={reading({capacity: 4})}/>)

        expect(pips().map(p => (p as HTMLElement).style.getPropertyValue("--click-budget-index")))
            .toEqual(["0", "1", "2", "3"])
    })

    it("warns when the wall is close", () => {
        render(<ClickBudgetMeter budget={reading({tokens: 2})}/>)
        expect(meter().classList.contains("click-budget-low")).toBe(true)
    })

    it("says so when there is nothing left", () => {
        render(<ClickBudgetMeter budget={reading({tokens: 0})}/>)

        expect(screen.getByText("0")).toBeTruthy()
        expect(meter().classList.contains("click-budget-empty")).toBe(true)
    })

    it("fades out when a full bucket has nothing to say", () => {
        render(<ClickBudgetMeter budget={reading({tokens: 10})}/>)

        expect(meter().classList.contains("click-budget-full")).toBe(true)
        expect(meter().classList.contains("click-budget-low")).toBe(false)
    })

    it("replays the refill between two readings instead of waiting for one", async () => {
        const start = performance.now()
        vi.spyOn(performance, "now").mockReturnValue(start)

        render(<ClickBudgetMeter budget={reading({tokens: 2, readAt: start})}/>)
        expect(screen.getByText("2")).toBeTruthy()

        vi.spyOn(performance, "now").mockReturnValue(start + 3_000)
        await vi.waitFor(() => expect(screen.getByText("5")).toBeTruthy())

        vi.restoreAllMocks()
    })
})

describe("ClickBudgetMeter while a bonus runs", () => {
    const running = (seconds = 60) => ({
        reward: {kind: "tripleClicks", seconds} as const,
        endsAt: performance.now() + seconds * 1000,
    })

    it("says nothing about a bonus when none is running", () => {
        render(<ClickBudgetMeter budget={reading()}/>)

        expect(document.querySelector(".click-budget-bonus")).toBeNull()
        expect(meter().classList.contains("click-budget-boosted")).toBe(false)
    })

    it("shows the multiplier that was won", () => {
        render(<ClickBudgetMeter budget={reading()} bonus={running()}/>)

        expect(screen.getByText("3×")).toBeTruthy()
    })

    it("counts down how long is left", () => {
        render(<ClickBudgetMeter budget={reading()} bonus={running(45)}/>)

        expect(screen.getByText("45s")).toBeTruthy()
    })

    it("marks the whole meter, so the boost reads at a glance", () => {
        render(<ClickBudgetMeter budget={reading()} bonus={running()}/>)

        expect(meter().classList.contains("click-budget-boosted")).toBe(true)
    })

    it("still reports the server's own allowance, never a multiplied guess", () => {
        // The boost is the server's to grant: when it does, capacity and rate
        // arrive in the reading and the pips widen on their own. Nothing here
        // may invent them in the meantime.
        render(<ClickBudgetMeter budget={reading({capacity: 10})} bonus={running()}/>)

        expect(pips()).toHaveLength(10)
        expect(meter().getAttribute("aria-valuemax")).toBe("10")
    })

    it("leaves the count itself alone", () => {
        render(<ClickBudgetMeter budget={reading({tokens: 4})} bonus={running()}/>)

        expect(meter().getAttribute("aria-valuenow")).toBe("4")
    })
})
