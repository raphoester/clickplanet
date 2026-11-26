// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {Shop, ShopRefusedError} from "../../backends/shop.ts"
import type {PlanPrice} from "../../domain/shop.ts"
import PlanOffer from "./PlanOffer.tsx"
import {useShop} from "./shopStore.ts"

const PLANS: PlanPrice[] = [
    {plan: "week", amount: 400, currency: "eur", renews: false},
    {plan: "month", amount: 1000, currency: "eur", renews: true},
    {plan: "year", amount: 6050, currency: "eur", renews: true},
]

const shopOf = (checkout: Shop["checkout"] = vi.fn(async () => "https://checkout.stripe.com/c/pay/cs_1")): Shop => ({
    catalog: vi.fn(async () => ({plans: PLANS})),
    checkout,
    myRenewals: vi.fn(async () => []),
    stopRenewal: vi.fn(),
})

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    useShop.setState(useShop.getInitialState(), true)
})

const plan = (name: RegExp) => screen.getByRole("radio", {name})

describe("PlanOffer", () => {
    it("shows nothing with no plans", () => {
        const {container} = render(<PlanOffer shop={shopOf()} plans={[]} target={{kind: "submission", id: "s-1"}} toCheckout={vi.fn()}/>)

        expect(container.textContent).toBe("")
        expect(screen.queryByRole("button")).toBeNull()
    })

    it("puts the plans side by side, each priced by the day, the best value picked", () => {
        render(<PlanOffer shop={shopOf()} plans={PLANS} target={{kind: "submission", id: "s-1"}} toCheckout={vi.fn()}/>)

        expect(screen.getAllByRole("radio").map((card) => card.textContent)).toEqual([
            "1 week€4Paid once€0.58 / day",
            "1 month€10Every month-42%€0.33 / day",
            "Best value1 year€60.50Every year-70%€0.17 / day",
        ])
        expect(plan(/1 year/).getAttribute("aria-checked")).toBe("true")
        expect(screen.getByRole("button", {name: "Pay €60.50"})).not.toBeNull()
        expect(screen.getByText("Secure payment by Stripe · Stop anytime")).not.toBeNull()
    })

    it("crosses out each price during a sale, shows the sale's under it with its cut and its clock, and Pays that", () => {
        const sold: PlanPrice[] = [PLANS[0], PLANS[1], {...PLANS[2], amount: 6000}]
        useShop.setState({sale: {percentOff: 50, endsAt: Date.now() + (2 * 24 * 3600 + 4 * 3600 + 12 * 60 + 9) * 1000}})
        render(<PlanOffer shop={shopOf()} plans={sold} target={{kind: "submission", id: "s-1"}} toCheckout={vi.fn()}/>)

        expect(screen.getAllByRole("radio").map((card) => card.textContent)).toEqual([
            "1 week€4€2Paid once-50%€0.29 / day",
            "1 month€10€5Then €10 a month-50%€0.17 / day",
            "Best value1 year€60€30Then €60 a year-50%€0.09 / day",
        ])
        expect(screen.getByRole("timer").textContent).toMatch(/^Black Friday · ends in 2d 04h 1[12]m \d\ds$/)
        expect(screen.getByRole("button", {name: "Pay €30"})).not.toBeNull()
    })

    it("asks for the consent before Pay, then sends the plan picked and the target and goes to Stripe's page", async () => {
        const checkout = vi.fn(async () => "https://checkout.stripe.com/c/pay/cs_1")
        const toCheckout = vi.fn()
        render(<PlanOffer shop={shopOf(checkout)} plans={PLANS} target={{kind: "submission", id: "s-1"}} toCheckout={toCheckout}/>)
        const user = userEvent.setup()

        await user.click(plan(/1 month/))
        const consent = screen.getByRole("checkbox", {name: "I want my flag to start as soon as it's approved. I know I can't cancel once it starts."})
        const pay = screen.getByRole("button", {name: "Pay €10"})
        expect(plan(/1 month/).getAttribute("aria-checked")).toBe("true")
        expect((consent as HTMLInputElement).checked).toBe(false)
        expect((pay as HTMLButtonElement).disabled).toBe(true)
        expect(screen.getByRole("link", {name: "Terms of sale"}).getAttribute("href")).toBe("/sale")
        expect(screen.getByRole("link", {name: "Refunds"}).getAttribute("target")).toBe("_blank")

        await user.click(consent)
        await user.click(pay)

        expect(checkout).toHaveBeenCalledWith({plan: "month", target: {kind: "submission", id: "s-1"}, startsNow: true})
        expect(toCheckout).toHaveBeenCalledWith("https://checkout.stripe.com/c/pay/cs_1")
    })

    it("keeps the plan picked for the next offer", async () => {
        const {unmount} = render(<PlanOffer shop={shopOf()} plans={PLANS} target={{kind: "submission", id: "s-1"}} toCheckout={vi.fn()}/>)
        await userEvent.setup().click(plan(/1 week/))
        unmount()

        render(<PlanOffer shop={shopOf()} plans={PLANS} target={{kind: "flag", flag: "r-k3p9x2ma0b"}} toCheckout={vi.fn()}/>)
        expect(plan(/1 week/).getAttribute("aria-checked")).toBe("true")
        expect(screen.getByText("Secure payment by Stripe")).not.toBeNull()
    })

    it("says a refusal in one line, and keeps the plan and the consent", async () => {
        const checkout = vi.fn(async (): Promise<string> => {
            throw new ShopRefusedError("unavailable")
        })
        const toCheckout = vi.fn()
        render(<PlanOffer shop={shopOf(checkout)} plans={PLANS} target={{kind: "flag", flag: "r-k3p9x2ma0b"}} toCheckout={toCheckout}/>)
        const user = userEvent.setup()

        await user.click(plan(/1 week/))
        await user.click(screen.getByRole("checkbox"))
        await user.click(screen.getByRole("button", {name: "Pay €4"}))

        expect(screen.getByRole("alert").textContent).toBe("You can't pay for this flag now.")
        expect(toCheckout).not.toHaveBeenCalled()
        expect(plan(/1 week/).getAttribute("aria-checked")).toBe("true")
        expect((screen.getByRole("checkbox") as HTMLInputElement).checked).toBe(true)
        expect((screen.getByRole("button", {name: "Pay €4"}) as HTMLButtonElement).disabled).toBe(false)
    })

    it("says any other failure as the payment page that did not open", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        render(<PlanOffer shop={shopOf(vi.fn(async (): Promise<string> => {
            throw new Error("offline")
        }))} plans={PLANS} target={{kind: "submission", id: "s-1"}} toCheckout={vi.fn()}/>)
        const user = userEvent.setup()

        await user.click(screen.getByRole("checkbox"))
        await user.click(screen.getByRole("button", {name: "Pay €60.50"}))

        expect(screen.getByRole("alert").textContent).toBe("The payment page did not open. Try again.")
    })

    it("goes back to what was there", async () => {
        const onBack = vi.fn()
        render(<PlanOffer shop={shopOf()} plans={PLANS} target={{kind: "flag", flag: "r-k3p9x2ma0b"}} toCheckout={vi.fn()} onBack={onBack}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Back"}))
        expect(onBack).toHaveBeenCalledTimes(1)
    })
})
