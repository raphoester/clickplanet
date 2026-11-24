// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {Shop} from "../../backends/shop.ts"
import type {PlanPrice} from "../../domain/shop.ts"
import {useFlagForm} from "../flags/flagFormStore.ts"
import {useFlagSubmissions} from "../flags/submissionsStore.ts"
import {useScreen} from "../hud/screenStore.ts"
import ShopModal from "./ShopModal.tsx"
import {useShop, useShopFeed} from "./shopStore.ts"

const PLANS: PlanPrice[] = [
    {plan: "week", amount: 400, currency: "eur", renews: false},
    {plan: "month", amount: 1000, currency: "eur", renews: true},
    {plan: "year", amount: 6000, currency: "eur", renews: true},
]

const shop = (changes: Partial<Shop> = {}): Shop => ({
    plans: vi.fn(async () => PLANS),
    checkout: vi.fn(async () => "https://checkout.stripe.com/c/pay/cs_1"),
    myRenewals: vi.fn(async () => []),
    stopRenewal: vi.fn(),
    ...changes,
})

afterEach(() => {
    cleanup()
    useScreen.setState(useScreen.getInitialState(), true)
    useShop.setState(useShop.getInitialState(), true)
    useFlagForm.setState(useFlagForm.getInitialState(), true)
    useFlagSubmissions.getState().serve(undefined)
})

describe("ShopModal", () => {
    it("shows nothing until the shop is opened", () => {
        render(<ShopModal/>)
        expect(screen.queryByRole("dialog")).toBeNull()
    })

    it("sells the region's flag at each plan's price, and opens the flag's place to order it", async () => {
        useShop.setState({plans: PLANS})
        useScreen.setState({shopOpen: true})
        render(<ShopModal/>)
        const user = userEvent.setup()

        expect(screen.getByRole("dialog", {name: "Shop"})).not.toBeNull()
        expect(screen.getByRole("heading", {name: "Your region's flag"})).not.toBeNull()
        const prices = screen.getAllByRole("listitem").filter((item) => item.classList.contains("shop-product-price"))
        expect(prices.map((item) => item.textContent)).toEqual(["1 week€4", "1 month€10 / month", "1 year€60 / year"])

        await user.click(screen.getByRole("button", {name: "Add your flag"}))
        expect(useScreen.getState().shopOpen).toBe(false)
        expect(useFlagForm.getState().shown).toBe(true)
    })

    it("closes on its close button", async () => {
        useShop.setState({plans: PLANS})
        useScreen.setState({shopOpen: true})
        render(<ShopModal/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Close"}))
        expect(useScreen.getState().shopOpen).toBe(false)
    })
})

describe("useShopFeed", () => {
    function Feed({from, signedIn}: {from: Shop, signedIn: boolean}) {
        useShopFeed(from, signedIn)
        return null
    }

    it("reads the plans at once for a guest too, and a guest's renewals never", async () => {
        const guest = shop()
        render(<Feed from={guest} signedIn={false}/>)

        await vi.waitFor(() => expect(useShop.getState().plans).toEqual(PLANS))
        act(() => useShop.getState().refresh())
        expect(guest.myRenewals).not.toHaveBeenCalled()
    })

    it("reads a signed-in account's renewals when asked", async () => {
        const player = shop()
        render(<Feed from={player} signedIn={true}/>)

        act(() => useShop.getState().refresh())
        await vi.waitFor(() => expect(player.myRenewals).toHaveBeenCalledTimes(1))
    })

    it("shows no plan while the shop is off", async () => {
        const off = shop({plans: vi.fn(async () => [])})
        render(<Feed from={off} signedIn={false}/>)

        await vi.waitFor(() => expect(off.plans).toHaveBeenCalled())
        expect(useShop.getState().plans).toEqual([])
    })
})
