// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import ClicksPanel from "./ClicksPanel.tsx"
import {ClickBudget} from "../../backends/clickBudget.ts"

afterEach(cleanup)

const TOLL = [{share: 0.1, slowdown: 1.5}, {share: 0.2, slowdown: 2.5}, {share: 0.3, slowdown: 4}]

function reading(overrides: Partial<ClickBudget> = {}): ClickBudget {
    return {tokens: 42, capacity: 60, perSecond: 0.05, readAt: performance.now(), ...overrides}
}

const steps = () => screen.getAllByRole("row").slice(1).map((row) => within(row).getAllByRole("cell").map((c) => c.textContent))
const here = () => screen.getAllByRole("row").filter((row) => row.getAttribute("aria-current") === "true")

describe("ClicksPanel", () => {
    it("shows the clicks in hand against the bank", () => {
        render(<ClicksPanel budget={reading()} countryName="France" toll={TOLL}/>)

        expect(screen.getByText("42")).toBeTruthy()
        expect(screen.getByText("of 60")).toBeTruthy()
    })

    it("lists every step of the toll, and marks the one the country is on", () => {
        render(<ClicksPanel budget={reading({price: {slowdown: 4, share: 0.98}})} countryName="France" toll={TOLL}/>)

        expect(screen.getByText("France holds 98% of the map")).toBeTruthy()
        expect(steps()).toEqual([
            ["Under 10%", "Plain rate"],
            ["10%", "1.5× slower"],
            ["20%", "2.5× slower"],
            ["30%", "4× slower"],
        ])
        expect(here().map((row) => within(row).getAllByRole("cell")[0].textContent)).toEqual(["30%"])
    })

    it("reads the share off the board when the server prices nothing", () => {
        render(<ClicksPanel budget={reading()} countryName="Bulgaria" share={0.001} toll={TOLL}/>)

        expect(screen.getByText("Bulgaria holds 0.1% of the map")).toBeTruthy()
        expect(here().map((row) => within(row).getAllByRole("cell")[0].textContent)).toEqual(["Under 10%"])
    })

    it("draws no table against a server that sends no steps", () => {
        render(<ClicksPanel budget={reading()} countryName="France" share={0.5} toll={[]}/>)

        expect(screen.queryByRole("table")).toBeNull()
    })

    it("says who else spends from the bank", () => {
        render(<ClicksPanel budget={reading({sharedWith: "network"})} countryName="France" toll={TOLL}/>)

        expect(screen.getByText("Shared with everyone on your network")).toBeTruthy()
    })

    it("offers a guest to click faster, at the server's multiplier", () => {
        const onSignIn = vi.fn()
        render(<ClicksPanel budget={reading({linkedMultiplier: 2})} countryName="France" toll={TOLL} onSignIn={onSignIn}/>)

        fireEvent.click(screen.getByRole("button", {name: "Sign in: clicks 2× faster"}))
        expect(onSignIn).toHaveBeenCalledTimes(1)
    })

    it("offers nothing to a player who is not a guest", () => {
        render(<ClicksPanel budget={reading({linkedMultiplier: 2})} countryName="France" toll={TOLL}/>)

        expect(screen.queryByRole("button")).toBeNull()
    })

    it("offers an ad for a full bank while the bank is not full, on CrazyGames only", () => {
        const onWatch = vi.fn()
        render(<ClicksPanel budget={reading()} countryName="France" toll={TOLL} ad={{state: {kind: "ready"}, onWatch}}/>)

        fireEvent.click(screen.getByRole("button", {name: "Watch an ad: full bank"}))

        expect(onWatch).toHaveBeenCalled()
        cleanup()
        render(<ClicksPanel budget={reading({tokens: 60})} countryName="France" toll={TOLL} ad={{state: {kind: "ready"}, onWatch}}/>)
        expect(screen.queryByRole("button", {name: "Watch an ad: full bank"})).toBeNull()
        cleanup()
        render(<ClicksPanel budget={reading()} countryName="France" toll={TOLL}/>)
        expect(screen.queryByRole("button", {name: "Watch an ad: full bank"})).toBeNull()
    })

    it("says when the next ad refill is, and that an adblocker has none", () => {
        render(<ClicksPanel budget={reading()} countryName="France" toll={TOLL} ad={{state: {kind: "waiting", until: Date.now() + 60_000}, onWatch: vi.fn()}}/>)
        expect(screen.getByText(/^Next ad refill at /)).toBeTruthy()
        cleanup()

        render(<ClicksPanel budget={reading()} countryName="France" toll={TOLL} ad={{state: {kind: "blocked"}, onWatch: vi.fn()}}/>)
        expect(screen.getByText(/^Ads are blocked/)).toBeTruthy()
    })
})
