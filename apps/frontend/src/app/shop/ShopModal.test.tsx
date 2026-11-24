// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {Shop} from "../../backends/shop.ts"
import {FlagCatalog} from "../../domain/flags.ts"
import type {Submission} from "../../domain/flagSubmissions.ts"
import type {PlanPrice, Renewal} from "../../domain/shop.ts"
import {useFlagForm} from "../flags/flagFormStore.ts"
import {useFlags} from "../flags/flagsStore.ts"
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
    useFlags.setState({catalog: FlagCatalog.BUILT_IN})
})

const FLAG = "r-k3p9x2ma0b"
const ENDS_AT = Date.UTC(2026, 11, 24)
const PICTURE = {url: "https://media.clickplanet.lol/flags/pictures/ab.png", width: 768, height: 512, stretch: false, focusX: 0.5, focusY: 0.5}

const submission = (changes: Partial<Submission> = {}): Submission => ({
    id: "s-1", name: "Bavaria", flag: "", state: "approved", submittedAt: 1, decidedAt: 2, ...changes,
})

function opened() {
    useFlags.setState({catalog: FlagCatalog.BUILT_IN.withFlags([{code: FLAG, name: "Bavaria", closed: false, sold: {picture: PICTURE, endsAt: ENDS_AT}}])})
    useFlagSubmissions.setState({submissions: [submission({state: "opened", flag: FLAG})], read: true})
}

function openShop(sells: Shop = shop(), toCheckout = vi.fn()) {
    useShop.setState({plans: PLANS})
    useScreen.setState({shopOpen: true})
    render(<ShopModal shop={sells} toCheckout={toCheckout}/>)
    return userEvent.setup()
}

const plan = (name: RegExp) => screen.getByRole("radio", {name})

describe("ShopModal", () => {
    it("shows nothing until the shop is opened", () => {
        render(<ShopModal shop={shop()} toCheckout={vi.fn()}/>)
        expect(screen.queryByRole("dialog")).toBeNull()
    })

    it("sells the region's flag with the plans side by side, and takes the plan picked to the flag's place", async () => {
        const user = openShop()

        expect(screen.getByRole("dialog", {name: "Shop"})).not.toBeNull()
        expect(screen.getByRole("heading", {name: "Your region's flag"})).not.toBeNull()
        expect(screen.getAllByRole("radio").map((card) => card.textContent)).toEqual([
            "1 week€4Paid once€0.58 / day",
            "1 month€10Every month-42%€0.33 / day",
            "Best value1 year€60Every year-71%€0.17 / day",
        ])
        expect(screen.getByRole("listitem", {current: "step"}).textContent).toBe("1Send your flag")

        await user.click(plan(/1 month/))
        await user.click(screen.getByRole("button", {name: "Add your flag"}))
        expect(useShop.getState().chosen).toBe("month")
        expect(useScreen.getState().shopOpen).toBe(false)
        expect(useFlagForm.getState().shown).toBe(true)
    })

    it("shows a flag in review at its second step, with no pay button", () => {
        useFlagSubmissions.setState({submissions: [submission({state: "waiting"})], read: true})
        openShop()

        expect(screen.getByRole("heading", {name: "Bavaria"})).not.toBeNull()
        expect(screen.getByRole("listitem", {current: "step"}).textContent).toBe("2We check it")
        expect(screen.queryByRole("button", {name: /^Pay/})).toBeNull()
        expect(screen.getByRole("button", {name: "Your flag"})).not.toBeNull()
    })

    it("pays for an approved flag from the shop itself", async () => {
        useFlagSubmissions.setState({submissions: [submission({picture: PICTURE})], read: true})
        const sells = shop()
        const toCheckout = vi.fn()
        const user = openShop(sells, toCheckout)

        expect(screen.getByRole("listitem", {current: "step"}).textContent).toBe("3Pay to open it")
        await user.click(screen.getByRole("checkbox"))
        await user.click(screen.getByRole("button", {name: "Pay €60"}))

        expect(sells.checkout).toHaveBeenCalledWith({plan: "year", target: {kind: "submission", id: "s-1"}, startsNow: true})
        expect(toCheckout).toHaveBeenCalledWith("https://checkout.stripe.com/c/pay/cs_1")
    })

    it("sells time for a flag the account founded, with no steps", async () => {
        opened()
        const sells = shop()
        const user = openShop(sells)

        expect(screen.getByRole("heading", {name: "Bavaria"})).not.toBeNull()
        expect(screen.queryByRole("listitem", {current: "step"})).toBeNull()
        await user.click(plan(/1 week/))
        await user.click(screen.getByRole("checkbox"))
        await user.click(screen.getByRole("button", {name: "Pay €4"}))

        expect(sells.checkout).toHaveBeenCalledWith({plan: "week", target: {kind: "flag", flag: FLAG}, startsNow: true})
    })

    it("sells no second plan while a flag's renewal runs", () => {
        opened()
        const renews: Renewal = {id: "r-1", plan: "month", target: {kind: "flag", flag: FLAG}, state: "renewing", periodEndsAt: ENDS_AT}
        useShop.setState({renewals: [renews]})
        openShop()

        expect(screen.queryByRole("radio")).toBeNull()
        expect(screen.queryByRole("button", {name: /^Pay/})).toBeNull()
        expect(screen.getByRole("button", {name: "Your flag"})).not.toBeNull()
    })

    it("closes on its close button", async () => {
        const user = openShop()

        await user.click(screen.getByRole("button", {name: "Close"}))
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
