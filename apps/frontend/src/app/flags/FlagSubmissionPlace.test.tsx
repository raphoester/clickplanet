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
import type {PlanPrice} from "../../domain/shop.ts"
import {useShop} from "../shop/shopStore.ts"
import {AccountStore} from "../account/accountStore.ts"
import {useFlagForm} from "./flagFormStore.ts"
import {useFlags} from "./flagsStore.ts"
import FlagSubmissionEntry from "./FlagSubmissionEntry.tsx"
import FlagSubmissionPlace from "./FlagSubmissionPlace.tsx"
import {useFlagSubmissions} from "./submissionsStore.ts"

const PIN = {latitude: 48.1, longitude: 11.6}

const submission = (changes: Partial<Submission> = {}): Submission => ({
    id: "s-1", name: "Bavaria", pin: PIN, flag: "", state: "waiting", submittedAt: 1, decidedAt: 1, paid: false, ...changes,
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

const png = () => new File(["png"], "bavaria.png", {type: "image/png"})

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
    it("shows nobody it, and a guest or a player the way to its region's flag, whatever it sent", () => {
        const view = render(<FlagSubmissionEntry/>)
        expect(screen.queryByRole("button")).toBeNull()

        act(() => useFlagSubmissions.setState({who: "guest"}))
        expect(screen.getByRole("button").textContent).toBe("Add your region's flag")

        act(() => {
            useFlagSubmissions.setState({who: "player"})
            read([submission({state: "opened", flag: FLAG})])
        })
        expect(screen.getByRole("button").textContent).toBe("Add your region's flag")
        expect(screen.getByRole("button").className).toContain("shop-shine")

        fireEvent.click(screen.getByRole("button"))
        expect(useFlagForm.getState()).toMatchObject({shown: true, step: {kind: "make"}, draft: {name: ""}})
        view.unmount()
    })

    it("names the flag as the picker's search typed it", () => {
        useFlagSubmissions.setState({who: "player"})
        read([])
        render(<FlagSubmissionEntry typed=" Bavaria"/>)

        expect(screen.getByRole("button").textContent).toBe("Put “Bavaria” on the map")
        fireEvent.click(screen.getByRole("button"))
        expect(useFlagForm.getState()).toMatchObject({shown: true, step: {kind: "make"}, draft: {name: "Bavaria"}})
    })
})

describe("FlagSubmissionPlace", () => {
    const place = (account: AccountStore, sells: Shop = shop(), toCheckout = vi.fn()) =>
        render(<FlagSubmissionPlace account={account} submitter={submitter()} shop={sells} toCheckout={toCheckout}/>)

    it("shows a guest the region's flag and the sign-in buttons, never the form", async () => {
        const account = await accountOf([])
        useFlagForm.getState().open()
        place(account)

        expect(screen.getByRole("dialog", {name: "Add your region's flag"})).not.toBeNull()
        expect(screen.getByText("Put your region on the map")).not.toBeNull()
        expect(screen.getByText("Sign in to add your flag.")).not.toBeNull()
        expect(screen.getByRole("button", {name: "Sign in with Google"})).not.toBeNull()
        expect(screen.queryByRole("textbox")).toBeNull()
    })

    it("opens a door on a new flag for an account with none on its way: none, rejected or founded", async () => {
        const account = await accountOf(["google"])
        for (const submissions of [[], [submission({state: "rejected"})], [submission({state: "opened", flag: FLAG})]]) {
            read(submissions)
            useFlagForm.getState().open()
            const view = place(account)
            expect(screen.getByRole("dialog", {name: "Add your region's flag"})).not.toBeNull()
            expect(screen.getByRole("textbox", {name: "Name"})).not.toBeNull()
            view.unmount()
        }
    })

    it("opens a door on the paywall of the flag on its way, sent or approved", async () => {
        const account = await accountOf(["google"])
        for (const sent of [submission(), submission({state: "approved"}), submission({paid: true})]) {
            read([sent, submission({id: "s-0", state: "opened", flag: FLAG})])
            useFlagForm.getState().open("Swabia")
            const view = place(account)
            expect(screen.getByRole("dialog", {name: "Your flag"})).not.toBeNull()
            expect(document.querySelector(".flag-card-name")?.textContent).toBe("Bavaria")
            view.unmount()
        }
    })

    it("waits for the account's submissions before it opens a door's step", async () => {
        const account = await accountOf(["google"])
        useFlagSubmissions.setState({submissions: [], read: false})
        useFlagForm.getState().open("Bavaria")
        place(account)
        expect(screen.getByRole("status").textContent).toBe("Loading…")

        act(() => read([]))
        expect((screen.getByRole("textbox", {name: "Name"}) as HTMLInputElement).value).toBe("Bavaria")
    })

    it("puts the form aside while the pin is aimed, and brings it back as it was on Escape or with the pin", async () => {
        const account = await accountOf(["google"])
        read([])
        useFlagForm.getState().compose({name: "Bavaria"})
        place(account)
        const user = userEvent.setup()

        await user.click(screen.getByRole("button", {name: "Put the capital on the globe"}))
        expect(screen.queryByRole("dialog")).toBeNull()
        expect(screen.getByText("Tap your region's capital")).not.toBeNull()

        await user.keyboard("{Escape}")
        expect((screen.getByRole("textbox", {name: "Name"}) as HTMLInputElement).value).toBe("Bavaria")

        await user.click(screen.getByRole("button", {name: "Put the capital on the globe"}))
        act(() => useFlagForm.getState().pinned({pin: PIN, place: "Upper Bavaria"}))
        expect(screen.getByText("Capital")).not.toBeNull()
        expect(screen.getAllByText("Upper Bavaria")).toHaveLength(2)
        expect(screen.getByText("48.10° N, 11.60° E")).not.toBeNull()
    })
})

describe("the paywall", () => {
    const typed = {name: "Bavaria", file: png(), pinned: {pin: PIN, place: "Upper Bavaria"}}

    async function paywall(sent: Submission, sells: Shop = shop(), toCheckout = vi.fn()) {
        const account = await accountOf(["google"])
        read([sent])
        useShop.setState({plans: PLANS})
        useFlagForm.setState({draft: typed})
        useFlagForm.getState().showPaywall(sent, typed)
        return render(<FlagSubmissionPlace account={account} submitter={submitter()} shop={sells} toCheckout={toCheckout}/>)
    }

    it("shows the flag made, the plans once, and pays before the verdict", async () => {
        const sells = shop()
        const toCheckout = vi.fn()
        await paywall(submission(), sells, toCheckout)
        const user = userEvent.setup()

        expect(document.querySelector(".flag-card")?.textContent).toBe("BavariaUpper BavariaWaiting")
        expect(document.querySelector(".flag-card image")?.getAttribute("href")).toBe("blob:preview")
        expect(screen.getByRole("status").textContent).toBe("We check it after you pay.")
        expect(screen.getAllByRole("radiogroup")).toHaveLength(1)
        expect(screen.getByRole("radio", {name: /1 year/}).getAttribute("aria-checked")).toBe("true")
        await user.click(screen.getByRole("checkbox"))
        await user.click(screen.getByRole("button", {name: "Pay €60"}))

        expect(sells.checkout).toHaveBeenCalledWith({plan: "year", target: {kind: "submission", id: "s-1"}, startsNow: true})
        expect(toCheckout).toHaveBeenCalledWith("https://checkout.stripe.com/c/pay/cs_1")
    })

    it("goes back to the flag while it waits, as a change of it", async () => {
        await paywall(submission())

        await userEvent.setup().click(screen.getByRole("button", {name: "Back"}))

        expect(screen.getByRole("dialog", {name: "Change your flag"})).not.toBeNull()
        expect(useFlagForm.getState().draft).toEqual({...typed, replaces: "s-1"})
    })

    it("shows an approval as it comes, which no Back changes, and keeps the plan and the consent", async () => {
        const sells = shop()
        await paywall(submission(), sells)
        const user = userEvent.setup()
        await user.click(screen.getByRole("checkbox"))

        act(() => read([submission({state: "approved", picture: PICTURE})]))

        expect(document.querySelector(".flag-card")?.textContent).toBe("BavariaUpper BavariaApproved")
        expect(document.querySelector(".flag-card image")?.getAttribute("href")).toBe(PICTURE.url)
        expect(screen.getByRole("status").textContent).toBe("Approved! Pick a plan to open it.")
        expect(screen.queryByRole("button", {name: "Back"})).toBeNull()
        await user.click(screen.getByRole("button", {name: "Pay €60"}))
        expect(sells.checkout).toHaveBeenCalledOnce()
    })

    it("shows a rejection before any payment, and fixes it from what was typed as a new flag", async () => {
        await paywall(submission())

        act(() => read([submission({state: "rejected", rejection: {reason: "not-a-place", note: "A club's crest."}})]))

        expect(screen.getByText("Rejected")).not.toBeNull()
        expect(screen.getByRole("status").textContent).toBe("It is not the flag of a place.")
        expect(screen.getByText("A club's crest.")).not.toBeNull()
        expect(screen.getByRole("link", {name: "Think we got it wrong? Ask a person to look again."}).getAttribute("href"))
            .toBe("mailto:support@clickplanet.lol?subject=Look%20again%20at%20my%20flag%20Bavaria")
        expect(screen.queryByRole("radio")).toBeNull()
        expect(screen.queryByRole("button", {name: /^Pay/})).toBeNull()

        await userEvent.setup().click(screen.getByRole("button", {name: "Fix it"}))

        expect(screen.getByRole("dialog", {name: "Add your region's flag"})).not.toBeNull()
        expect(useFlagForm.getState().draft).toEqual(typed)
    })

    it("fixes a rejection found by a door from what was sent, the picture picked again", async () => {
        const account = await accountOf(["google"])
        read([submission({state: "approved"})])
        useFlagForm.getState().open()
        render(<FlagSubmissionPlace account={account} submitter={submitter()} shop={shop()} toCheckout={vi.fn()}/>)

        act(() => read([submission({state: "rejected", rejection: {reason: "other", note: ""}})]))
        await userEvent.setup().click(screen.getByRole("button", {name: "Fix it"}))

        expect(useFlagForm.getState().draft).toEqual({name: "Bavaria", pinned: {pin: PIN}})
    })

    it("offers nothing more and no change for a flag paid, while it is checked and as it opens", async () => {
        await paywall(submission({paid: true}))

        expect(screen.getByRole("status").textContent).toBe("Paid! We check your flag, then it opens.")
        expect(screen.queryByRole("radio")).toBeNull()
        expect(screen.queryByRole("button", {name: "Back"})).toBeNull()

        act(() => read([submission({state: "approved", paid: true, picture: PICTURE})]))
        expect(screen.getByRole("status").textContent).toBe("Approved! It opens now.")
        expect(screen.queryByRole("radio")).toBeNull()

        act(() => read([submission({state: "opened", flag: FLAG, picture: PICTURE})]))
        expect(screen.getByRole("status").textContent).toBe("Bavaria is in the game!")
    })

    it("offers no plan while the shop is off, and goes back to a flag that waits", async () => {
        await paywall(submission({submittedAt: Date.UTC(2026, 10, 24, 12)}))
        act(() => useShop.setState({plans: []}))

        expect(screen.getByRole("status").textContent).toBe("Sent on Nov 24, 2026.")
        expect(screen.queryByRole("radio")).toBeNull()
        expect(document.querySelector(".modal-footer")?.textContent).toBe("Back")

        act(() => read([submission({state: "approved"})]))
        expect(screen.getByRole("status").textContent).toBe("Approved! It opens soon.")
        expect(document.querySelector(".modal-footer")).toBeNull()
    })

    it("shows the flag as sent while no read lists it yet", async () => {
        await paywall(submission({id: "s-2", name: "Swabia"}))
        act(() => read([]))

        expect(document.querySelector(".flag-card-name")?.textContent).toBe("Swabia")
    })
})
