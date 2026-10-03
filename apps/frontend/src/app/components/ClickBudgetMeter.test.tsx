// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from 'vitest'
import {cleanup, fireEvent, render, screen} from '@testing-library/react'
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
    const slowed = {slowdown: 4, share: 0.98}

    it("says the refill is slower for a country that holds much of the map", () => {
        render(<ClickBudgetMeter budget={reading({price: slowed})}/>)

        expect(screen.getByRole("img", {name: "Refills 4× slower"}).textContent).toBe("4× slower")
    })

    it("says it in short on a phone", () => {
        render(<ClickBudgetMeter compact budget={reading({price: slowed})}/>)

        expect(screen.getByRole("img", {name: "Refills 4× slower"}).textContent).toBe("4×")
    })

    it("keeps the slowdown out of the meter, which is a reading", () => {
        render(<ClickBudgetMeter budget={reading({price: slowed})}/>)

        expect(meter().querySelector(".click-budget-toll")).toBeNull()
    })

    it("says nothing about price at the plain rate", () => {
        render(<ClickBudgetMeter budget={reading({price: {slowdown: 1, share: 0.01, next: {share: 0.1, slowdown: 2}}})}/>)

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

    it("leaves the offer to the clicks panel when the line already says the refill is slower", () => {
        render(<ClickBudgetMeter budget={reading({linkedMultiplier: 2, price: {slowdown: 4, share: 0.98}})} onSignIn={vi.fn()}/>)

        expect(screen.queryByRole("button", {name: /Sign in/})).toBeNull()
    })

    it("leaves the offer to the clicks panel on a phone", () => {
        render(<ClickBudgetMeter compact budget={reading({linkedMultiplier: 2})} onSignIn={vi.fn()}/>)

        expect(screen.queryByRole("button", {name: /Sign in/})).toBeNull()
    })

    it("offers a guest who shares its bank one of its own", () => {
        render(<ClickBudgetMeter budget={reading({linkedMultiplier: 2, sharedWith: "guests"})} onSignIn={vi.fn()}/>)

        expect(screen.getByRole("button", {name: "Sign in: your own clicks"})).toBeTruthy()
    })
})

describe("ClickBudgetMeter's shared bucket", () => {
    it("says nothing about a player's own bucket", () => {
        render(<ClickBudgetMeter budget={reading()}/>)

        expect(document.querySelector(".click-budget-shared")).toBeNull()
    })

    it("says who else spends from it", () => {
        const {rerender} = render(<ClickBudgetMeter budget={reading({sharedWith: "guests"})}/>)
        expect(screen.getByRole("img", {name: "Shared with the guests on your network"})).toBeTruthy()

        rerender(<ClickBudgetMeter budget={reading({sharedWith: "network"})}/>)
        expect(screen.getByRole("img", {name: "Shared with everyone on your network"})).toBeTruthy()
    })
})

describe("ClickBudgetMeter's dock", () => {
    it("holds what it is given under the meter", () => {
        render(<ClickBudgetMeter budget={reading()}><p>held</p></ClickBudgetMeter>)

        expect(document.querySelector(".click-budget-dock")?.textContent).toContain("held")
    })

    it("puts it after the reading, in the one panel", () => {
        render(<ClickBudgetMeter budget={reading()}><p>held</p></ClickBudgetMeter>)

        const dock = document.querySelector(".click-budget-dock")!
        const children = Array.from(dock.children)
        expect(children.indexOf(document.querySelector(".click-budget-shell")!))
            .toBeLessThan(children.indexOf(screen.getByText("held")))
    })

    it("still holds it against a server that reports no allowance", () => {
        render(<ClickBudgetMeter><p>held</p></ClickBudgetMeter>)

        expect(screen.getByText("held")).toBeTruthy()
        expect(screen.queryByRole("meter")).toBeNull()
    })
})

describe("ClickBudgetMeter and your clicks", () => {
    it("opens them from the reading", () => {
        const onToggleOpen = vi.fn()
        render(<ClickBudgetMeter budget={reading()} onToggleOpen={onToggleOpen}/>)

        const open = screen.getByRole("button", {name: "Your clicks"})
        expect(open.getAttribute("aria-expanded")).toBe("false")
        expect(meter().contains(open)).toBe(false)

        fireEvent.click(open)
        expect(onToggleOpen).toHaveBeenCalledTimes(1)
    })

    it("shows them over the dock on a desktop, and closes them on Escape", () => {
        const onToggleOpen = vi.fn()
        render(<ClickBudgetMeter budget={reading()} open onToggleOpen={onToggleOpen} popover={<p>steps</p>}/>)

        expect(screen.getByText("steps")).toBeTruthy()
        expect(screen.getByRole("button", {name: "Your clicks"}).getAttribute("aria-expanded")).toBe("true")

        fireEvent.keyDown(document, {key: "Escape"})
        expect(onToggleOpen).toHaveBeenCalledTimes(1)
    })

    it("leaves them to a sheet on a phone", () => {
        render(<ClickBudgetMeter compact budget={reading()} open onToggleOpen={vi.fn()} popover={<p>steps</p>}/>)

        expect(screen.queryByText("steps")).toBeNull()
    })
})
