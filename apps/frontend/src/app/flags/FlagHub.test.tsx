// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {Shop} from "../../backends/shop.ts"
import {Held, Submission} from "../../domain/flagSubmissions.ts"
import {FlagCatalog, SoldFlag} from "../../domain/flags.ts"
import type {PlanPrice, Renewal} from "../../domain/shop.ts"
import {translator} from "../../i18n/page.ts"
import {useShop} from "../shop/shopStore.ts"
import {useDismissedFlagRows} from "./dismissedFlagRows.ts"
import {useFlagForm} from "./flagFormStore.ts"
import FlagHub from "./FlagHub.tsx"
import FlagPayScreen from "./FlagPayScreen.tsx"
import {useFlags} from "./flagsStore.ts"
import {FlagShooter} from "./mapShot.ts"
import {useFlagSubmissions} from "./submissionsStore.ts"

const PIN = {latitude: 48.1, longitude: 11.6}
const FLAG = "r-k3p9x2ma0b"
const PICTURE = {url: "https://media.clickplanet.lol/flags/pictures/ab.png", width: 768, height: 512, stretch: false, focusX: 0.5, focusY: 0.5}
const ENDS_AT = Date.UTC(2026, 11, 24)
const CLOSED_AT = Date.UTC(2026, 10, 24, 12)

const PLANS: PlanPrice[] = [
    {plan: "week", amount: 400, currency: "eur", renews: false},
    {plan: "month", amount: 1000, currency: "eur", renews: true},
    {plan: "year", amount: 6000, currency: "eur", renews: true},
]

// The globe is in no test: its shot never comes.
const NEVER_SHOT: FlagShooter = () => new Promise(() => {})

const submission = (changes: Partial<Submission> = {}): Submission => ({
    id: "s-1", name: "Bavaria", pin: PIN, flag: "", state: "waiting", submittedAt: 1, decidedAt: 1, paid: false, ...changes,
})

const shop = (changes: Partial<Shop> = {}): Shop => ({
    catalog: vi.fn(async () => ({plans: []})),
    checkout: vi.fn(async () => "https://checkout.stripe.com/c/pay/cs_1"),
    myRenewals: vi.fn(async () => []),
    stopRenewal: vi.fn(),
    ...changes,
})

const dateOf = (at: number) => translator().dateTime(at, {dateStyle: "medium"})

const read = (submissions: Submission[], held: Held[] = []) => useFlagSubmissions.setState({submissions, held, read: true})

function flagIs(state: "open" | "closed" | "removed") {
    const sold: SoldFlag = {
        picture: PICTURE, endsAt: ENDS_AT, pin: PIN,
        ...state === "open" ? {} : {closedAt: CLOSED_AT},
        ...state === "removed" ? {removed: true} : {},
    }
    useFlags.setState({catalog: FlagCatalog.BUILT_IN.withFlags([{code: FLAG, name: "Bavaria", closed: state !== "open", sold}])})
}

const rows = () => [...document.querySelectorAll(".flag-row")].map((row) => row.textContent)

afterEach(() => {
    cleanup()
    useFlagForm.setState(useFlagForm.getInitialState(), true)
    useFlagSubmissions.setState(useFlagSubmissions.getInitialState(), true)
    useShop.setState(useShop.getInitialState(), true)
    useFlags.setState({catalog: FlagCatalog.BUILT_IN})
    useDismissedFlagRows.setState({dismissed: new Set()})
    window.localStorage.clear()
    vi.restoreAllMocks()
})

describe("the account's flags", () => {
    it("list each flag held, then each flag sent that did not open, under a new flag", () => {
        flagIs("open")
        useShop.setState({plans: PLANS})
        read([
            submission({id: "s-3", name: "Swabia"}),
            submission({id: "s-2", name: "Franconia", state: "rejected", rejection: {reason: "unreadable", note: "Too small."}}),
            submission({state: "opened", flag: FLAG}),
        ], [{code: FLAG}])
        render(<FlagHub shop={shop()}/>)

        expect(screen.getByRole("dialog", {name: "Your flags"})).not.toBeNull()
        expect(screen.getByRole("button", {name: "Add a new flag"})).not.toBeNull()
        expect(rows()).toEqual([
            `BavariaEnds on ${dateOf(ENDS_AT)}Add time`,
            "SwabiaWe check it after you pay.Pay",
            "FranconiaIt is hard to read at the size of a tile.Too small.Think we got it wrong? Ask a person to look again.",
        ])
        expect(document.querySelector(".chip"), "the line says the state").toBeNull()
        expect(screen.getByText("Too small.")).not.toBeNull()
        expect(screen.getByRole("link", {name: "Think we got it wrong? Ask a person to look again."}).getAttribute("href"))
            .toBe("mailto:support@clickplanet.lol?subject=Look%20again%20at%20my%20flag%20Franconia")
    })

    it("open a new flag in the place of the one that waits unpaid", async () => {
        read([submission({name: "Sardinia"})])
        render(<FlagHub shop={shop()}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Add a new flag"}))

        expect(useFlagForm.getState()).toMatchObject({
            step: {kind: "make"}, draft: {name: "", replaces: "s-1", replacing: "Sardinia"},
        })
    })

    it("give each flag sent one action: Pay while the shop sells, Change while it is off", async () => {
        useShop.setState({plans: PLANS})
        const approved = submission({id: "s-2", name: "Swabia", state: "approved", picture: PICTURE})
        read([submission(), approved])
        const view = render(<FlagHub shop={shop()}/>)
        const user = userEvent.setup()

        expect(screen.queryByRole("button", {name: "Change"}), "its paywall changes it").toBeNull()
        await user.click(screen.getAllByRole("button", {name: "Pay"})[1])
        expect(useFlagForm.getState()).toMatchObject({step: {kind: "pay", submission: approved}})

        view.unmount()
        useShop.setState({plans: []})
        render(<FlagHub shop={shop()}/>)
        await user.click(screen.getByRole("button", {name: "Change"}))
        expect(useFlagForm.getState()).toMatchObject({step: {kind: "make"}, draft: {name: "Bavaria", pinned: {pin: PIN}, replaces: "s-1"}})
    })

    it("show a flag sent once: the newest of a name, and none of a name the account holds", () => {
        flagIs("closed")
        read([
            submission({id: "s-4", name: "kurdistan", state: "rejected", rejection: {reason: "exists", note: ""}}),
            submission({id: "s-3", name: "Kurdistan", state: "rejected", rejection: {reason: "hateful", note: ""}}),
            submission({id: "s-2", name: "Bavaria", state: "rejected", rejection: {reason: "exists", note: ""}}),
            submission({state: "opened", flag: FLAG}),
        ], [{code: FLAG}])
        render(<FlagHub shop={shop()}/>)

        expect([...document.querySelectorAll(".flag-row-name")].map((name) => name.textContent)).toEqual(["Bavaria", "kurdistan"])
    })

    it("put away a row that is done, for good in this browser, and never one that waits", async () => {
        flagIs("removed")
        read([
            submission({id: "s-3", name: "Swabia"}),
            submission({id: "s-2", name: "Sardinia", state: "approved"}),
        ], [{code: FLAG, removalNote: "A club's crest."}])
        const view = render(<FlagHub shop={shop()}/>)
        const user = userEvent.setup()

        expect(screen.queryByRole("button", {name: "Hide Swabia"})).toBeNull()
        await user.click(screen.getByRole("button", {name: "Hide Sardinia"}))
        await user.click(screen.getByRole("button", {name: "Hide Bavaria"}))
        expect([...document.querySelectorAll(".flag-row-name")].map((name) => name.textContent)).toEqual(["Swabia"])

        view.unmount()
        expect(JSON.parse(window.localStorage.getItem("clickplanet-flag-rows-dismissed") ?? "[]")).toEqual(["s-2", `flag:${FLAG}`])
    })

    it("bring back a flag held once it is open again, though it was put away removed", async () => {
        flagIs("removed")
        read([], [{code: FLAG, removalNote: "A club's crest."}])
        render(<FlagHub shop={shop()}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Hide Bavaria"}))
        expect(rows()).toEqual([])

        act(() => flagIs("open"))
        expect([...document.querySelectorAll(".flag-row-name")].map((name) => name.textContent)).toEqual(["Bavaria"])
    })

    it("open the pay page of a flag held to add time, never the plans in the list", async () => {
        flagIs("closed")
        useShop.setState({plans: PLANS})
        read([], [{code: FLAG}])
        render(<FlagHub shop={shop()}/>)

        expect(rows()[0]).toBe(`BavariaClosed on ${dateOf(CLOSED_AT)}Add time`)
        await userEvent.setup().click(screen.getByRole("button", {name: "Add time"}))

        expect(useFlagForm.getState().step).toEqual({kind: "flag", flag: FLAG, fromHub: true})
        expect(screen.queryByRole("radio")).toBeNull()
    })

    it("say when a renewal pays next in place of Add time, and stop it, which brings Add time back", async () => {
        flagIs("open")
        read([submission({state: "opened", flag: FLAG})], [{code: FLAG}])
        const renews: Renewal = {id: "r-1", plan: "month", target: {kind: "submission", id: "s-1"}, state: "renewing", periodEndsAt: ENDS_AT}
        useShop.setState({plans: PLANS, renewals: [renews]})
        const sells = shop({stopRenewal: vi.fn(async () => ({...renews, state: "stopping" as const}))})
        render(<FlagHub shop={sells}/>)
        const user = userEvent.setup()

        expect(screen.getByText(`Renews on ${dateOf(ENDS_AT)}`)).not.toBeNull()
        expect(screen.queryByRole("button", {name: "Add time"})).toBeNull()
        await user.click(screen.getByRole("button", {name: "Stop renewal"}))

        expect(sells.stopRenewal).toHaveBeenCalledWith("r-1")
        expect(screen.getByText(`Ends on ${dateOf(ENDS_AT)}`)).not.toBeNull()
        expect(screen.getByRole("button", {name: "Add time"}), "a renewal stopping charges no more").not.toBeNull()
    })

    it("say a renewal that did not stop in one line, and keep it", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        flagIs("open")
        read([], [{code: FLAG}])
        const renews: Renewal = {id: "r-1", plan: "year", target: {kind: "flag", flag: FLAG}, state: "renewing", periodEndsAt: ENDS_AT}
        useShop.setState({plans: PLANS, renewals: [renews]})
        render(<FlagHub shop={shop({stopRenewal: vi.fn(async () => Promise.reject(new Error("offline")))})}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Stop renewal"}))

        expect(screen.getByRole("alert").textContent).toBe("The renewal did not stop. Try again.")
    })

    it("show a removed flag with the day and the operator's note, and nothing to pay", () => {
        flagIs("removed")
        useShop.setState({plans: PLANS})
        read([], [{code: FLAG, removalNote: "A club's crest."}])
        render(<FlagHub shop={shop()}/>)

        expect(rows()[0]).toBe(`BavariaRemoved on ${dateOf(CLOSED_AT)}A club's crest.`)
        expect(screen.queryByRole("button", {name: "Add time"})).toBeNull()
    })

    it("show no time and no Add time while the shop is off", () => {
        flagIs("open")
        read([], [{code: FLAG}])
        render(<FlagHub shop={shop()}/>)

        expect(screen.queryByRole("button", {name: "Add time"})).toBeNull()
        expect(screen.queryByText(/^Ends on/)).toBeNull()
    })

    it("leave a flag another player claimed out", () => {
        flagIs("open")
        read([submission({state: "opened", flag: FLAG})], [])
        render(<FlagHub shop={shop()}/>)

        expect(rows()).toEqual([])
    })
})

describe("a flag's pay page", () => {
    it("adds time to a flag held through the plans, and goes back to the account's flags", async () => {
        flagIs("open")
        useShop.setState({plans: PLANS})
        read([], [{code: FLAG}])
        const sells = shop()
        render(<FlagPayScreen flag={FLAG} fromHub shooter={NEVER_SHOT} shop={sells} toCheckout={vi.fn()}/>)
        const user = userEvent.setup()

        expect(screen.getByRole("dialog", {name: "Add time"})).not.toBeNull()
        expect(document.querySelector(".flag-card-name")?.textContent).toBe("Bavaria")
        expect(screen.getByRole("status").textContent).toBe(`Ends on ${dateOf(ENDS_AT)}`)
        await user.click(screen.getByRole("radio", {name: /1 week/}))
        await user.click(screen.getByRole("checkbox"))
        await user.click(screen.getByRole("button", {name: "Pay €4"}))
        expect(sells.checkout).toHaveBeenCalledWith({plan: "week", target: {kind: "flag", flag: FLAG}, startsNow: true})
    })

    it("goes back to the account's flags", async () => {
        flagIs("open")
        useShop.setState({plans: PLANS})
        read([], [{code: FLAG}])
        render(<FlagPayScreen flag={FLAG} fromHub shooter={NEVER_SHOT} shop={shop()} toCheckout={vi.fn()}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Back"}))

        expect(useFlagForm.getState().step).toEqual({kind: "hub"})
    })

    it("claims a flag whose time ended, for the flag, and offers nothing once it opened for another", async () => {
        flagIs("closed")
        useShop.setState({plans: PLANS})
        read([], [])
        const sells = shop()
        render(<FlagPayScreen flag={FLAG} fromHub={false} shooter={NEVER_SHOT} shop={sells} toCheckout={vi.fn()}/>)
        const user = userEvent.setup()

        expect(screen.getByRole("dialog", {name: "Claim Bavaria"})).not.toBeNull()
        await user.click(screen.getByRole("radio", {name: /1 month/}))
        await user.click(screen.getByRole("checkbox"))
        await user.click(screen.getByRole("button", {name: "Pay €10"}))
        expect(sells.checkout).toHaveBeenCalledWith({plan: "month", target: {kind: "flag", flag: FLAG}, startsNow: true})

        act(() => flagIs("open"))
        expect(screen.queryByRole("radio")).toBeNull()
    })

    it("offers nothing for a removed flag", () => {
        flagIs("removed")
        useShop.setState({plans: PLANS})
        read([], [])
        render(<FlagPayScreen flag={FLAG} fromHub={false} shooter={NEVER_SHOT} shop={shop()} toCheckout={vi.fn()}/>)

        expect(screen.queryByRole("radio")).toBeNull()
        expect(screen.getByText("Removed")).not.toBeNull()
    })
})
