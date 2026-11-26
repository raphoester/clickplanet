// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {Shop} from "../../backends/shop.ts"
import {Submission} from "../../domain/flagSubmissions.ts"
import {FlagCatalog} from "../../domain/flags.ts"
import type {PlanPrice, Renewal} from "../../domain/shop.ts"
import {translator} from "../../i18n/page.ts"
import {useShop} from "../shop/shopStore.ts"
import {useFlagForm} from "./flagFormStore.ts"
import {useFlags} from "./flagsStore.ts"
import {useFlagSubmissions} from "./submissionsStore.ts"
import YourFlags from "./YourFlags.tsx"

const PIN = {latitude: 48.1, longitude: 11.6}
const FLAG = "r-k3p9x2ma0b"
const PICTURE = {url: "https://media.clickplanet.lol/flags/pictures/ab.png", width: 768, height: 512, stretch: false, focusX: 0.5, focusY: 0.5}
const ENDS_AT = Date.UTC(2026, 11, 24)

const PLANS: PlanPrice[] = [
    {plan: "week", amount: 400, currency: "eur", renews: false},
    {plan: "month", amount: 1000, currency: "eur", renews: true},
    {plan: "year", amount: 6000, currency: "eur", renews: true},
]

const submission = (changes: Partial<Submission> = {}): Submission => ({
    id: "s-1", name: "Bavaria", pin: PIN, flag: "", state: "waiting", submittedAt: 1, decidedAt: 1, paid: false, ...changes,
})

const shop = (changes: Partial<Shop> = {}): Shop => ({
    plans: vi.fn(async () => []),
    checkout: vi.fn(async () => "https://checkout.stripe.com/c/pay/cs_1"),
    myRenewals: vi.fn(async () => []),
    stopRenewal: vi.fn(),
    ...changes,
})

const dateOf = (at: number) => translator().dateTime(at, {dateStyle: "medium"})

const read = (submissions: Submission[]) => useFlagSubmissions.setState({submissions, read: true})

function flagIs(closed: boolean) {
    const sold = {picture: PICTURE, endsAt: ENDS_AT, ...closed ? {closedAt: Date.UTC(2026, 10, 24, 12)} : {}}
    useFlags.setState({catalog: FlagCatalog.BUILT_IN.withFlags([{code: FLAG, name: "Bavaria", closed, sold}])})
}

// The flag the account founded, open in the list of flags.
function founded(changes: Partial<Submission> = {}) {
    flagIs(false)
    read([submission({state: "opened", flag: FLAG, ...changes})])
}

afterEach(() => {
    cleanup()
    useFlagForm.setState(useFlagForm.getInitialState(), true)
    useFlagSubmissions.setState(useFlagSubmissions.getInitialState(), true)
    useShop.setState(useShop.getInitialState(), true)
    useFlags.setState({catalog: FlagCatalog.BUILT_IN})
    vi.restoreAllMocks()
})

describe("YourFlags", () => {
    it("shows nothing for an account that sent no flag", () => {
        read([])
        render(<YourFlags shop={shop()} toCheckout={vi.fn()}/>)

        expect(screen.queryByText("Your flags")).toBeNull()
    })

    it("lists each flag sent with its state, the newest first", () => {
        read([
            submission({id: "s-3", name: "Swabia"}),
            submission({id: "s-2", name: "Franconia", state: "rejected", rejection: {reason: "unreadable", note: "Too small."}}),
            submission({state: "opened", flag: FLAG}),
        ])
        flagIs(false)
        render(<YourFlags shop={shop()} toCheckout={vi.fn()}/>)

        expect(screen.getByRole("heading", {name: "Your flags"})).not.toBeNull()
        expect([...document.querySelectorAll(".flag-founded")].map((row) => row.querySelector(".chip")?.textContent))
            .toEqual(["Waiting", "Rejected", "Open"])
        expect(screen.getByText("It is hard to read at the size of a tile.")).not.toBeNull()
        expect(screen.getByText("Too small.")).not.toBeNull()
        expect(screen.getByRole("link", {name: "Think we got it wrong? Ask a person to look again."}).getAttribute("href"))
            .toBe("mailto:support@clickplanet.lol?subject=Look%20again%20at%20my%20flag%20Franconia")
    })

    it("changes a flag that waits unpaid from what was sent, out of the profile", async () => {
        read([submission()])
        const away = vi.fn()
        render(<YourFlags shop={shop()} toCheckout={vi.fn()} onAway={away}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Change"}))

        expect(away).toHaveBeenCalledOnce()
        expect(useFlagForm.getState()).toMatchObject({
            shown: true,
            step: {kind: "make"},
            draft: {name: "Bavaria", pinned: {pin: PIN}, replaces: "s-1"},
        })
    })

    it("changes no flag once paid or decided, and pays for one approved on its paywall", async () => {
        useShop.setState({plans: PLANS})
        const approved = submission({id: "s-2", state: "approved", picture: PICTURE})
        read([approved, submission({paid: true})])
        render(<YourFlags shop={shop()} toCheckout={vi.fn()}/>)

        expect(screen.queryByRole("button", {name: "Change"})).toBeNull()
        await userEvent.setup().click(screen.getByRole("button", {name: "Pay"}))

        expect(useFlagForm.getState()).toMatchObject({shown: true, step: {kind: "pay", submission: approved}})
    })

    it("adds time to a founded flag through the plans, for the flag", async () => {
        founded()
        useShop.setState({plans: PLANS})
        const sells = shop()
        render(<YourFlags shop={sells} toCheckout={vi.fn()}/>)
        const user = userEvent.setup()

        expect(screen.getByText(`Ends on ${dateOf(ENDS_AT)}`)).not.toBeNull()
        await user.click(screen.getByRole("button", {name: "Add time"}))
        await user.click(screen.getByRole("button", {name: "Back"}))
        await user.click(screen.getByRole("button", {name: "Add time"}))
        await user.click(screen.getByRole("radio", {name: /1 week/}))
        await user.click(screen.getByRole("checkbox"))
        await user.click(screen.getByRole("button", {name: "Pay €4"}))

        expect(sells.checkout).toHaveBeenCalledWith({plan: "week", target: {kind: "flag", flag: FLAG}, startsNow: true})
    })

    it("shows no time and no Add time while the shop is off", () => {
        founded()
        render(<YourFlags shop={shop()} toCheckout={vi.fn()}/>)

        expect(screen.queryByRole("button", {name: "Change"}), "a flag approved never changes").toBeNull()
        expect(screen.queryByRole("button", {name: "Add time"})).toBeNull()
        expect(screen.queryByText(/^Ends on/)).toBeNull()
    })

    it("says when a renewal pays next in place of Add time, and stops it, which brings Add time back", async () => {
        founded()
        const renews: Renewal = {id: "r-1", plan: "month", target: {kind: "submission", id: "s-1"}, state: "renewing", periodEndsAt: ENDS_AT}
        useShop.setState({plans: PLANS, renewals: [renews]})
        const sells = shop({stopRenewal: vi.fn(async () => ({...renews, state: "stopping" as const}))})
        render(<YourFlags shop={sells} toCheckout={vi.fn()}/>)
        const user = userEvent.setup()

        expect(screen.getByText(`Renews on ${dateOf(ENDS_AT)}`)).not.toBeNull()
        expect(screen.queryByRole("button", {name: "Add time"})).toBeNull()
        await user.click(screen.getByRole("button", {name: "Stop renewal"}))

        expect(sells.stopRenewal).toHaveBeenCalledWith("r-1")
        expect(screen.getByText(`Ends on ${dateOf(ENDS_AT)}`)).not.toBeNull()
        expect(screen.queryByRole("button", {name: "Stop renewal"})).toBeNull()
        expect(screen.getByRole("button", {name: "Add time"}), "a renewal stopping charges no more").not.toBeNull()
    })

    it("says a renewal that did not stop in one line, and keeps it", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        founded()
        const renews: Renewal = {id: "r-1", plan: "year", target: {kind: "flag", flag: FLAG}, state: "renewing", periodEndsAt: ENDS_AT}
        useShop.setState({plans: PLANS, renewals: [renews]})
        render(<YourFlags shop={shop({stopRenewal: vi.fn(async () => Promise.reject(new Error("offline")))})} toCheckout={vi.fn()}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Stop renewal"}))

        expect(screen.getByRole("alert").textContent).toBe("The renewal did not stop. Try again.")
        expect(screen.getByText(`Renews on ${dateOf(ENDS_AT)}`)).not.toBeNull()
    })

    it("marks a founded flag closed, with the day and the operator's note, until it comes back", () => {
        founded({closeNote: "A club's crest."})
        useShop.setState({plans: PLANS})
        render(<YourFlags shop={shop()} toCheckout={vi.fn()}/>)
        expect(document.querySelector(".flag-founded")?.textContent).toBe(`BavariaEnds on ${dateOf(ENDS_AT)}OpenAdd time`)

        act(() => flagIs(true))
        expect(document.querySelector(".flag-founded")?.textContent)
            .toBe(`BavariaClosed on ${dateOf(Date.UTC(2026, 10, 24, 12))}ClosedAdd timeA club's crest.`)

        act(() => flagIs(false))
        expect(document.querySelector(".flag-status-note")).toBeNull()
    })

    it("folds into the phone's You sheet as its other sections", () => {
        read([submission()])
        render(<YourFlags shop={shop()} toCheckout={vi.fn()} folded/>)

        expect(screen.getByRole("button", {name: "Your flags"}).getAttribute("aria-expanded")).toBe("true")
    })
})
