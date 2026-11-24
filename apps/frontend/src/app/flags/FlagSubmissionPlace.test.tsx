// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {AccountBackend, Provider} from "../../backends/account.ts"
import {FlagSubmitter} from "../../backends/flags.ts"
import {NameColor, PlayerBackend} from "../../backends/player.ts"
import {ProgressionBackend} from "../../backends/progression.ts"
import {Submission} from "../../domain/flagSubmissions.ts"
import {AccountStore} from "../account/accountStore.ts"
import {useFlagForm} from "./flagFormStore.ts"
import FlagSubmissionEntry from "./FlagSubmissionEntry.tsx"
import FlagSubmissionPlace from "./FlagSubmissionPlace.tsx"
import {useFlagSubmissions} from "./submissionsStore.ts"

const PIN = {latitude: 48.1, longitude: 11.6}

const submission = (changes: Partial<Submission> = {}): Submission => ({
    id: "s-1", name: "Bavaria", pin: PIN, change: false, flag: "", state: "waiting", submittedAt: 1, decidedAt: 1, ...changes,
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
        render(<FlagSubmissionPlace account={account} submitter={submitter()}/>)

        expect(screen.getByRole("dialog", {name: "Add a flag"})).not.toBeNull()
        expect(screen.getByText("Sign in to add your flag.")).not.toBeNull()
        expect(screen.getByRole("button", {name: "Sign in with Google"})).not.toBeNull()
        expect(screen.queryByRole("textbox")).toBeNull()
    })

    it("opens on the form for an account with no submission", async () => {
        const account = await accountOf(["google"])
        read([])
        useFlagForm.getState().show()
        render(<FlagSubmissionPlace account={account} submitter={submitter()}/>)

        expect(screen.getByRole("dialog", {name: "Add a flag"})).not.toBeNull()
        expect(screen.getByRole("textbox", {name: "Name"})).not.toBeNull()
    })

    it("shows a submission that waits, with no way to send another", async () => {
        const account = await accountOf(["google"])
        read([submission()])
        useFlagForm.getState().show()
        render(<FlagSubmissionPlace account={account} submitter={submitter()}/>)

        expect(screen.getByRole("dialog", {name: "Your flag"})).not.toBeNull()
        expect(screen.getByText("Waiting")).not.toBeNull()
        expect(screen.queryByRole("button", {name: "New flag"})).toBeNull()
        expect(document.querySelector(".modal-footer")?.textContent).toBe("Close")
    })

    it("shows a rejection's reason and the operator's note, and sends again from what was sent", async () => {
        const account = await accountOf(["google"])
        read([submission({state: "rejected", rejection: {reason: "not-a-place", note: "A club's crest."}})])
        useFlagForm.getState().show()
        render(<FlagSubmissionPlace account={account} submitter={submitter()}/>)

        expect(screen.getByText("It is not the flag of a place.")).not.toBeNull()
        expect(screen.getByText("A club's crest.")).not.toBeNull()

        await userEvent.setup().click(screen.getByRole("button", {name: "Try again"}))

        expect((screen.getByRole("textbox", {name: "Name"}) as HTMLInputElement).value).toBe("Bavaria")
        expect(useFlagForm.getState().draft.pinned).toEqual({pin: PIN})
    })

    it("says an approval, an opening, and offers a change of the flag the account founded", async () => {
        const account = await accountOf(["google"])
        read([submission({state: "approved"})])
        useFlagForm.getState().show()
        const view = render(<FlagSubmissionPlace account={account} submitter={submitter()}/>)
        expect(screen.getByText("Approved! It opens soon.")).not.toBeNull()

        act(() => read([submission({state: "opened", flag: "r-k3p9x2ma0b"})]))
        expect(screen.getByRole("status").textContent).toBe("Bavaria is in the game!")

        await userEvent.setup().click(screen.getByRole("button", {name: "Change"}))
        expect(screen.getByRole("dialog", {name: "Change your flag"})).not.toBeNull()
        expect(useFlagForm.getState().draft.flag).toBe("r-k3p9x2ma0b")
        view.unmount()
    })

    it("puts the form aside while the pin is aimed, and brings it back as it was on Escape or with the pin", async () => {
        const account = await accountOf(["google"])
        read([])
        useFlagForm.getState().compose({name: "Bavaria"})
        render(<FlagSubmissionPlace account={account} submitter={submitter()}/>)
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
})
