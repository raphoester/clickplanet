// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {Shop} from "../../backends/shop.ts"
import type {PlanPrice} from "../../domain/shop.ts"
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
    useShop.setState(useShop.getInitialState(), true)
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
