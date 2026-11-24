// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {AccountBackend, Provider} from "../../backends/account.ts"
import {FlagSubmitter} from "../../backends/flags.ts"
import {Shop} from "../../backends/shop.ts"
import {NameColor, PlayerBackend} from "../../backends/player.ts"
import {ProgressionBackend} from "../../backends/progression.ts"
import {Submission} from "../../domain/flagSubmissions.ts"
import {FlagCatalog} from "../../domain/flags.ts"
import type {PlanPrice, Renewal} from "../../domain/shop.ts"
import {translator} from "../../i18n/page.ts"
import {useShop} from "../shop/shopStore.ts"
import {AccountStore} from "../account/accountStore.ts"
import {useFlagForm} from "./flagFormStore.ts"
import {useFlags} from "./flagsStore.ts"
import FlagSubmissionEntry from "./FlagSubmissionEntry.tsx"
import FlagSubmissionPlace from "./FlagSubmissionPlace.tsx"
import {useFlagSubmissions} from "./submissionsStore.ts"

const PIN = {latitude: 48.1, longitude: 11.6}

const submission = (changes: Partial<Submission> = {}): Submission => ({
    id: "s-1", name: "Bavaria", pin: PIN, flag: "", state: "waiting", submittedAt: 1, decidedAt: 1, ...changes,
})

async function accountOf(linked: Provider[]) {
    const backend = {
        signInOptions: vi.fn(async (): Promise<Provider[]> => ["google", "discord"]),
        me: vi.fn(async () => ({linked})),
        startSignIn: vi.fn(async () => "https://accounts.example/authorize"),
        completeSignIn: vi.fn(async () => "signedIn" as const),
        startEmailSignIn: vi.fn(async () => undefined),
        completeEmailSignIn: vi.fn(async () => "signedIn" as const),
        signOut: vi.fn(async () => undefined),
        signOutEverywhere: vi.fn(async () => undefined),
        deleteAccount: vi.fn(async () => undefined),
    } satisfies AccountBackend
    const player = {
        profile: vi.fn(async () => ({accountId: "account-1", name: "Nova", color: NameColor.UNSPECIFIED})),
        setName: vi.fn(async (name: string) => ({accountId: "account-1", name})),
        setColor: vi.fn(async (color: NameColor) => color),
    } satisfies PlayerBackend
    const progression = {
        titles: vi.fn(async () => ({wearable: [], tracks: []})),
        fronts: vi.fn(async () => ({playsFor: [], playsAgainst: []})),
        wearTitle: vi.fn(async () => undefined),
    } satisfies ProgressionBackend
    const store = new AccountStore(backend, player, progression, {token: vi.fn(), held: vi.fn(), identity: vi.fn(), heldIdentity: vi.fn(), invalidate: vi.fn()},
        {navigate: vi.fn(), remember: vi.fn(), claimFlag: vi.fn()})
    await store.load()
    return store
}

const submitter = (): FlagSubmitter => ({uploadPicture: vi.fn(), submitFlag: vi.fn(), mySubmissions: vi.fn(async () => [])})

const shop = (changes: Partial<Shop> = {}): Shop => ({
    plans: vi.fn(async () => []),
    checkout: vi.fn(async () => "https://checkout.stripe.com/c/pay/cs_1"),
    myRenewals: vi.fn(async () => []),
    stopRenewal: vi.fn(),
    ...changes,
})

const PLANS: PlanPrice[] = [
    {plan: "week", amount: 400, currency: "eur", renews: false},
    {plan: "month", amount: 1000, currency: "eur", renews: true},
    {plan: "year", amount: 6000, currency: "eur", renews: true},
]

const FLAG = "r-k3p9x2ma0b"
const PICTURE = {url: "https://media.clickplanet.lol/flags/pictures/ab.png", width: 768, height: 512, stretch: false, focusX: 0.5, focusY: 0.5}
const ENDS_AT = Date.UTC(2026, 11, 24)

// The flag the account founded, open in the list of flags.
function founded() {
    useFlags.setState({catalog: FlagCatalog.BUILT_IN.withFlags([{code: FLAG, name: "Bavaria", closed: false, sold: {picture: PICTURE, endsAt: ENDS_AT}}])})
    read([submission({state: "opened", flag: FLAG})])
}

const dateOf = (at: number) => translator().dateTime(at, {dateStyle: "medium"})

// The store as a read answered it.
function read(submissions: Submission[]) {
    useFlagSubmissions.setState({submissions, read: true})
}

beforeEach(() => {
    URL.createObjectURL = vi.fn(() => "blob:preview")
    URL.revokeObjectURL = vi.fn()
})

afterEach(() => {
    cleanup()
    useFlagForm.setState(useFlagForm.getInitialState(), true)
    useFlagSubmissions.getState().serve(undefined)
    useFlagSubmissions.setState({who: "nobody"})
    useShop.setState(useShop.getInitialState(), true)
    useFlags.setState(useFlags.getInitialState(), true)
    useFlags.setState({catalog: FlagCatalog.BUILT_IN})
    vi.restoreAllMocks()
})

describe("FlagSubmissionEntry", () => {
    it("shows nobody it, a guest an entry to add a flag, and a player its latest submission and its state", () => {
        const view = render(<FlagSubmissionEntry/>)
        expect(screen.queryByRole("button")).toBeNull()

        act(() => useFlagSubmissions.setState({who: "guest"}))
        expect(screen.getByRole("button").textContent).toBe("Add a flag")

        act(() => {
            useFlagSubmissions.setState({who: "player"})
            read([submission({state: "rejected"})])
        })
        expect(screen.getByRole("button").textContent).toBe("BavariaRejected")

        fireEvent.click(screen.getByRole("button"))
        expect(useFlagForm.getState().shown).toBe(true)
        view.unmount()
    })
})

describe("FlagSubmissionPlace", () => {
    it("shows a guest the sign-in buttons, never the form", async () => {
        const account = await accountOf([])
        useFlagForm.getState().show()
        render(<FlagSubmissionPlace account={account} submitter={submitter()} shop={shop()} toCheckout={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "Add a flag"})).not.toBeNull()
        expect(screen.getByText("Sign in to add your flag.")).not.toBeNull()
        expect(screen.getByRole("button", {name: "Sign in with Google"})).not.toBeNull()
        expect(screen.queryByRole("textbox")).toBeNull()
    })

    it("opens on the form for an account with no submission", async () => {
        const account = await accountOf(["google"])
        read([])
        useFlagForm.getState().show()
        render(<FlagSubmissionPlace account={account} submitter={submitter()} shop={shop()} toCheckout={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "Add a flag"})).not.toBeNull()
        expect(screen.getByRole("textbox", {name: "Name"})).not.toBeNull()
    })

    it("shows a submission that waits, with no way to send another, and changes it from what was sent", async () => {
        const account = await accountOf(["google"])
        read([submission()])
        useFlagForm.getState().show()
        render(<FlagSubmissionPlace account={account} submitter={submitter()} shop={shop()} toCheckout={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "Your flag"})).not.toBeNull()
        expect(screen.getByText("Waiting")).not.toBeNull()
        expect(screen.queryByRole("button", {name: "New flag"})).toBeNull()
        expect(document.querySelector(".modal-footer")?.textContent).toBe("Change")

        await userEvent.setup().click(screen.getByRole("button", {name: "Change"}))
        expect(screen.getByRole("dialog", {name: "Change your flag"})).not.toBeNull()
        expect((screen.getByRole("textbox", {name: "Name"}) as HTMLInputElement).value).toBe("Bavaria")
        expect(useFlagForm.getState().draft).toEqual({name: "Bavaria", pinned: {pin: PIN}, replaces: "s-1"})
    })

    it("shows a rejection's reason and the operator's note, and sends again from what was sent", async () => {
        const account = await accountOf(["google"])
        read([submission({state: "rejected", rejection: {reason: "not-a-place", note: "A club's crest."}})])
        useFlagForm.getState().show()
        render(<FlagSubmissionPlace account={account} submitter={submitter()} shop={shop()} toCheckout={vi.fn()}/>)

        expect(screen.getByText("It is not the flag of a place.")).not.toBeNull()
        expect(screen.getByText("A club's crest.")).not.toBeNull()

        await userEvent.setup().click(screen.getByRole("button", {name: "Try again"}))

        expect((screen.getByRole("textbox", {name: "Name"}) as HTMLInputElement).value).toBe("Bavaria")
        expect(useFlagForm.getState().draft.pinned).toEqual({pin: PIN})
    })

    it("says an approval and an opening, and offers no change once the operator decided", async () => {
        const account = await accountOf(["google"])
        read([submission({state: "approved"})])
        useFlagForm.getState().show()
        const view = render(<FlagSubmissionPlace account={account} submitter={submitter()} shop={shop()} toCheckout={vi.fn()}/>)
        expect(screen.getByText("Approved! It opens soon.")).not.toBeNull()
        expect(screen.queryByRole("button", {name: "Change"})).toBeNull()

        act(() => read([submission({state: "opened", flag: "r-k3p9x2ma0b"})]))
        expect(screen.getByRole("status").textContent).toBe("Bavaria is in the game!")
        expect(screen.getByText("Your flags")).not.toBeNull()
        expect(screen.queryByRole("button", {name: "Change"})).toBeNull()
        view.unmount()
    })

    it("says until when the flag the account opened plays, and once closed, the day it closed", async () => {
        const account = await accountOf(["google"])
        const picture = {url: "https://media.example/bavaria.png", width: 300, height: 200, stretch: false, focusX: 0.5, focusY: 0.5}
        const sold = {picture, endsAt: Date.UTC(2026, 11, 24, 12), pin: PIN}
        const flagsAre = (closed: boolean) => useFlags.setState({
            catalog: FlagCatalog.BUILT_IN.withFlags([{
                code: "r-k3p9x2ma0b", name: "Bavaria", closed,
                sold: closed ? {...sold, closedAt: Date.UTC(2026, 10, 24, 12)} : sold,
            }]),
        })
        flagsAre(false)
        read([submission({state: "opened", flag: "r-k3p9x2ma0b"})])
        useFlagForm.getState().show()
        render(<><FlagSubmissionEntry/><FlagSubmissionPlace account={account} submitter={submitter()} shop={shop()} toCheckout={vi.fn()}/></>)
        act(() => useFlagSubmissions.setState({who: "player"}))

        expect(screen.getByRole("status").textContent).toBe("Bavaria is in the game until Dec 24, 2026!")
        expect(document.querySelector(".flag-entry")?.textContent).toBe("BavariaOpen")

        act(() => flagsAre(true))
        expect(screen.getByRole("status").textContent).toBe("Bavaria closed on Nov 24, 2026.")
        expect(document.querySelector(".flag-entry")?.textContent).toBe("BavariaClosed")
        expect(document.querySelector(".flag-founded")?.textContent).toBe("BavariaClosed")

        act(() => read([submission({state: "opened", flag: "r-k3p9x2ma0b", closeNote: "A club's crest."})]))
        expect(document.querySelector(".flag-status > .flag-status-note")?.textContent).toBe("A club's crest.")
        expect(document.querySelector(".flag-founded")?.textContent).toBe("BavariaClosed")

        act(() => read([
            submission({id: "s-2", name: "Swabia"}),
            submission({state: "opened", flag: "r-k3p9x2ma0b", closeNote: "A club's crest."}),
        ]))
        expect(document.querySelector(".flag-status > .flag-status-note")).toBeNull()
        expect(document.querySelector(".flag-founded")?.textContent).toBe("BavariaClosedA club's crest.")

        act(() => flagsAre(false))
        expect(document.querySelector(".flag-status-note")).toBeNull()
    })

    it("puts the form aside while the pin is aimed, and brings it back as it was on Escape or with the pin", async () => {
        const account = await accountOf(["google"])
        read([])
        useFlagForm.getState().compose({name: "Bavaria"})
        render(<FlagSubmissionPlace account={account} submitter={submitter()} shop={shop()} toCheckout={vi.fn()}/>)
        const user = userEvent.setup()

        await user.click(screen.getByRole("button", {name: "Put the capital on the globe"}))
        expect(screen.queryByRole("dialog")).toBeNull()
        expect(screen.getByText("Tap your region's capital")).not.toBeNull()

        await user.keyboard("{Escape}")
        expect((screen.getByRole("textbox", {name: "Name"}) as HTMLInputElement).value).toBe("Bavaria")

        await user.click(screen.getByRole("button", {name: "Put the capital on the globe"}))
        act(() => useFlagForm.getState().pinned({pin: PIN, place: "Upper Bavaria"}))
        expect(screen.getByText("Capital")).not.toBeNull()
        expect(screen.getByText("Upper Bavaria")).not.toBeNull()
        expect(screen.getByText("48.10° N, 11.60° E")).not.toBeNull()
    })

    it("offers no plan for an approval while the shop is off", async () => {
        const account = await accountOf(["google"])
        read([submission({state: "approved"})])
        useFlagForm.getState().show()
        render(<FlagSubmissionPlace account={account} submitter={submitter()} shop={shop()} toCheckout={vi.fn()}/>)

        expect(screen.getByText("Approved! It opens soon.")).not.toBeNull()
        expect(screen.queryByRole("button", {name: /^1 week/})).toBeNull()
        expect(screen.queryByRole("button", {name: "Pay"})).toBeNull()
    })

    it("offers the plans for an approved new flag, and pays for its submission", async () => {
        const account = await accountOf(["google"])
        read([submission({state: "approved"})])
        useShop.setState({plans: PLANS})
        useFlagForm.getState().show()
        const sells = shop()
        const toCheckout = vi.fn()
        render(<FlagSubmissionPlace account={account} submitter={submitter()} shop={sells} toCheckout={toCheckout}/>)
        const user = userEvent.setup()

        expect(screen.getByText("Approved! Pick a plan to open it.")).not.toBeNull()
        await user.click(screen.getByRole("button", {name: /^1 year/}))
        await user.click(screen.getByRole("checkbox"))
        await user.click(screen.getByRole("button", {name: "Pay"}))

        expect(sells.checkout).toHaveBeenCalledWith({plan: "year", target: {kind: "submission", id: "s-1"}, startsNow: true})
        expect(toCheckout).toHaveBeenCalledWith("https://checkout.stripe.com/c/pay/cs_1")
    })

    it("adds time to a founded flag through the same plans, for the flag", async () => {
        const account = await accountOf(["google"])
        founded()
        useShop.setState({plans: PLANS})
        useFlagForm.getState().show()
        const sells = shop()
        render(<FlagSubmissionPlace account={account} submitter={submitter()} shop={sells} toCheckout={vi.fn()}/>)
        const user = userEvent.setup()

        expect(screen.getByText(`Ends on ${dateOf(ENDS_AT)}`)).not.toBeNull()
        await user.click(screen.getByRole("button", {name: "Add time"}))
        await user.click(screen.getByRole("button", {name: /^1 week/}))
        await user.click(screen.getByRole("checkbox"))
        await user.click(screen.getByRole("button", {name: "Pay"}))

        expect(sells.checkout).toHaveBeenCalledWith({plan: "week", target: {kind: "flag", flag: FLAG}, startsNow: true})
    })

    it("shows no time and no Add time while the shop is off", async () => {
        const account = await accountOf(["google"])
        founded()
        useFlagForm.getState().show()
        render(<FlagSubmissionPlace account={account} submitter={submitter()} shop={shop()} toCheckout={vi.fn()}/>)

        expect(screen.queryByRole("button", {name: "Change"}), "a flag approved never changes").toBeNull()
        expect(screen.queryByRole("button", {name: "Add time"})).toBeNull()
        expect(screen.queryByText(/^Ends on/)).toBeNull()
    })

    it("says when a renewal pays next in place of Add time, and stops it, which brings Add time back", async () => {
        const account = await accountOf(["google"])
        founded()
        const renews: Renewal = {id: "r-1", plan: "month", target: {kind: "submission", id: "s-1"}, state: "renewing", periodEndsAt: ENDS_AT}
        useShop.setState({plans: PLANS, renewals: [renews]})
        useFlagForm.getState().show()
        const sells = shop({stopRenewal: vi.fn(async () => ({...renews, state: "stopping" as const}))})
        render(<FlagSubmissionPlace account={account} submitter={submitter()} shop={sells} toCheckout={vi.fn()}/>)
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
        const account = await accountOf(["google"])
        founded()
        const renews: Renewal = {id: "r-1", plan: "year", target: {kind: "flag", flag: FLAG}, state: "renewing", periodEndsAt: ENDS_AT}
        useShop.setState({plans: PLANS, renewals: [renews]})
        useFlagForm.getState().show()
        render(<FlagSubmissionPlace account={account}
                                    submitter={submitter()}
                                    shop={shop({stopRenewal: vi.fn(async () => Promise.reject(new Error("offline")))})}
                                    toCheckout={vi.fn()}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Stop renewal"}))

        expect(screen.getByRole("alert").textContent).toBe("The renewal did not stop. Try again.")
        expect(screen.getByText(`Renews on ${dateOf(ENDS_AT)}`)).not.toBeNull()
    })
})
