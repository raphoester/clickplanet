// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from 'vitest'
import {cleanup, fireEvent, render, screen} from '@testing-library/react'
import ClickBudgetMeter from './ClickBudgetMeter.tsx'
import {NO_CHARGES} from "../../domain/bonus.ts"
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

    it("says when the next click comes back when it is slow to", () => {
        render(<ClickBudgetMeter budget={reading({tokens: 0, capacity: 60, perSecond: 0.2})}/>)

        expect(screen.getByText("+1 in 5s")).toBeTruthy()
    })

    it("counts down with clicks in hand too", () => {
        render(<ClickBudgetMeter budget={reading({tokens: 20.5, capacity: 60, perSecond: 0.2})}/>)

        expect(screen.getByText("+1 in 3s")).toBeTruthy()
    })

    it("says nothing about the next click at a full bucket", () => {
        render(<ClickBudgetMeter budget={reading({tokens: 60, capacity: 60, perSecond: 0.2})}/>)

        expect(document.querySelector(".click-budget-next")?.textContent).toBe("")
    })

    it("has no countdown when a click comes back every second", () => {
        render(<ClickBudgetMeter budget={reading({tokens: 0})}/>)

        expect(document.querySelector(".click-budget-next")).toBeNull()
    })

    it("fills a strip under a long bar once per click", () => {
        render(<ClickBudgetMeter budget={reading({tokens: 20.5, capacity: 60, perSecond: 0.2})}/>)

        expect(Number(meter().style.getPropertyValue("--click-budget-next"))).toBeCloseTo(0.5, 1)
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

describe("ClickBudgetMeter and the price", () => {
    it("says why the refill is slow for a country that holds much of the map", () => {
        render(<ClickBudgetMeter countryName="Bulgaria"
                                 budget={reading({price: {slowdown: 8, share: 0.8, next: {share: 0.9, slowdown: 10}}})}/>)

        expect(screen.getByText("Bulgaria holds 80% of the map")).toBeTruthy()
        expect(screen.getByText("Refills 8× slower · 10× at 90%")).toBeTruthy()
    })

    it("says nothing about price at the plain rate", () => {
        render(<ClickBudgetMeter countryName="Chad" budget={reading({price: {slowdown: 1, share: 0.01, next: {share: 0.1, slowdown: 2}}})}/>)

        expect(document.querySelector(".click-budget-toll")).toBeNull()
    })

    it("shakes on each refused click, and not before", () => {
        const {rerender} = render(<ClickBudgetMeter budget={reading({tokens: 0})}/>)
        expect(meter().classList.contains("click-budget-refused")).toBe(false)

        rerender(<ClickBudgetMeter budget={reading({tokens: 0})} refusals={1}/>)
        expect(meter().classList.contains("click-budget-refused")).toBe(true)

        fireEvent.animationEnd(meter())
        expect(meter().classList.contains("click-budget-refused")).toBe(false)

        rerender(<ClickBudgetMeter budget={reading({tokens: 0})} refusals={2}/>)
        expect(meter().classList.contains("click-budget-refused")).toBe(true)
    })
})

describe("ClickBudgetMeter for a guest", () => {
    it("offers to click faster by signing in, at the server's multiplier", () => {
        const onSignIn = vi.fn()
        render(<ClickBudgetMeter budget={reading({linkedMultiplier: 2})} onSignIn={onSignIn}/>)

        fireEvent.click(screen.getByRole("button", {name: "Sign in: clicks 2× faster"}))

        expect(onSignIn).toHaveBeenCalledTimes(1)
    })

    it("keeps the offer out of the meter itself, which is a reading", () => {
        render(<ClickBudgetMeter budget={reading({linkedMultiplier: 2})} onSignIn={vi.fn()}/>)

        expect(meter().querySelector("button")).toBeNull()
    })

    it("offers nothing to a player who is not a guest", () => {
        render(<ClickBudgetMeter budget={reading({linkedMultiplier: 2})}/>)

        expect(screen.queryByRole("button")).toBeNull()
    })

    it("offers nothing when the server grants nothing for signing in", () => {
        render(<ClickBudgetMeter budget={reading()} onSignIn={vi.fn()}/>)

        expect(screen.queryByRole("button")).toBeNull()
    })
})

describe("ClickBudgetMeter with charges held", () => {
    it("says nothing when nothing is held", () => {
        render(<ClickBudgetMeter budget={reading()} charges={NO_CHARGES}/>)

        expect(document.querySelector(".click-budget-charges")).toBeNull()
    })

    it("says each charge held, with no countdown", () => {
        render(<ClickBudgetMeter budget={reading()}
                                 charges={{refill: false, bomb: true, enclose: true, spreadClicksLeft: 5}}
                                 onToggleBomb={() => {}}/>)

        expect(screen.getByText("Bomb ready")).toBeTruthy()
        expect(screen.getByText("Enclose ready")).toBeTruthy()
        expect(screen.getByText("Spread: 5 clicks left")).toBeTruthy()
        expect(screen.queryByText(/\ds$/)).toBeNull()
    })

    it("does not mark the meter boosted: a charge widens nothing", () => {
        render(<ClickBudgetMeter budget={reading()} charges={{...NO_CHARGES, spreadClicksLeft: 8}}/>)

        expect(meter().classList.contains("click-budget-boosted")).toBe(false)
    })

    it("aims the bomb and puts it away from its own button", () => {
        const toggle = vi.fn()
        const {rerender} = render(<ClickBudgetMeter budget={reading()}
                                                    charges={{...NO_CHARGES, bomb: true}}
                                                    onToggleBomb={toggle}/>)

        const bomb = screen.getByRole("button", {name: /Bomb ready/})
        expect(bomb.getAttribute("aria-pressed")).toBe("false")
        fireEvent.click(bomb)
        expect(toggle).toHaveBeenCalledTimes(1)

        rerender(<ClickBudgetMeter budget={reading()}
                                   charges={{...NO_CHARGES, bomb: true}}
                                   bombArmed
                                   onToggleBomb={toggle}/>)

        const aimed = screen.getByRole("button", {name: /hold to drop/})
        expect(aimed.getAttribute("aria-pressed")).toBe("true")
    })

    it("only says the bomb when there is no way to aim it", () => {
        render(<ClickBudgetMeter budget={reading()} charges={{...NO_CHARGES, bomb: true}}/>)

        expect(screen.getByText("Bomb ready")).toBeTruthy()
        expect(screen.queryByRole("button", {name: /Bomb/})).toBeNull()
    })
})

describe("ClickBudgetMeter with a refill held", () => {
    const refill = {...NO_CHARGES, refill: true}

    it("fills the bank on a press", () => {
        const use = vi.fn()
        render(<ClickBudgetMeter budget={reading({tokens: 4})} charges={refill} onUseRefill={use}/>)

        fireEvent.click(screen.getByRole("button", {name: /Refill ready/}))

        expect(use).toHaveBeenCalledTimes(1)
    })

    it("sends nothing on a full bank, and says so", () => {
        const use = vi.fn()
        render(<ClickBudgetMeter budget={reading({tokens: 10, capacity: 10})} charges={refill} onUseRefill={use}/>)

        fireEvent.click(screen.getByRole("button", {name: /Refill ready/}))

        expect(use).not.toHaveBeenCalled()
        expect(screen.getByRole("button", {name: /Bank already full/})).toBeTruthy()
    })

    it("only says the refill when there is no way to use it", () => {
        render(<ClickBudgetMeter budget={reading()} charges={refill}/>)

        expect(screen.getByText("Refill ready")).toBeTruthy()
        expect(screen.queryByRole("button", {name: /Refill/})).toBeNull()
    })
})
