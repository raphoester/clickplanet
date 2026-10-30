// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import AdRefillModal from "./AdRefillModal.tsx"
import {AdRefillState} from "./useAdRefill.ts"

afterEach(cleanup)

function modal(state: AdRefillState) {
    const adRefill = {state, watch: vi.fn(), seen: vi.fn()}
    const onClose = vi.fn()
    render(<AdRefillModal adRefill={adRefill} countryId="fr" onClose={onClose}/>)
    return {adRefill, onClose}
}

describe("AdRefillModal", () => {
    it("offers the ad and a way out that look the same", async () => {
        const {adRefill, onClose} = modal({kind: "ready"})
        const watch = screen.getByRole("button", {name: "Watch an ad: full bank"})
        const decline = screen.getByRole("button", {name: "No thanks"})
        expect(watch.className).toBe(decline.className)

        await userEvent.click(watch)
        expect(adRefill.watch).toHaveBeenCalledWith("fr")

        await userEvent.click(decline)
        expect(onClose).toHaveBeenCalled()
    })

    it("cannot be closed while the ad plays", () => {
        modal({kind: "playing"})

        expect((screen.getByRole("button", {name: "Watch an ad: full bank"}) as HTMLButtonElement).disabled).toBe(true)
        expect((screen.getByRole("button", {name: "No thanks"}) as HTMLButtonElement).disabled).toBe(true)
    })

    it("shows the reward, and the clock starts once it is seen", async () => {
        const {adRefill, onClose} = modal({kind: "refilled"})
        expect(screen.getByText("Your bank is full. Thanks for watching!")).toBeDefined()

        await userEvent.click(screen.getByRole("button", {name: "Play"}))

        expect(adRefill.seen).toHaveBeenCalled()
        expect(onClose).toHaveBeenCalled()
    })

    it("offers no ad that cannot play", () => {
        modal({kind: "blocked"})

        expect(screen.queryByRole("button", {name: "Watch an ad: full bank"})).toBeNull()
        expect(screen.getByText(/Ads are blocked/)).toBeDefined()
    })
})
