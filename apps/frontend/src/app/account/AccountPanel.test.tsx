// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {useSyncExternalStore} from "react"
import {AccountBackend, Provider} from "../../backends/account.ts"
import {ColoredProfile, NameColor, PlayerBackend, Profile, RenameTooSoonError} from "../../backends/player.ts"
import {AccountStore} from "./accountStore.ts"
import AccountPanel from "./AccountPanel.tsx"

afterEach(cleanup)

const DAY_MS = 24 * 3600_000

async function signedIn(profile: ColoredProfile, setName: (name: string) => Promise<Profile>) {
    const backend = {
        signInOptions: vi.fn(async (): Promise<Provider[]> => ["google"]),
        me: vi.fn(async (): Promise<{linked: Provider[]}> => ({linked: ["google"]})),
        startSignIn: vi.fn(async () => "https://google.example/authorize"),
        completeSignIn: vi.fn(async () => "signedIn" as const),
        startEmailSignIn: vi.fn(async () => undefined),
        completeEmailSignIn: vi.fn(async () => "signedIn" as const),
        signOut: vi.fn(async () => undefined),
        signOutEverywhere: vi.fn(async () => undefined),
        deleteAccount: vi.fn(async () => undefined),
    } satisfies AccountBackend
    const player = {
        profile: vi.fn(async () => profile),
        setName: vi.fn(setName),
        setColor: vi.fn(async (color: NameColor) => color),
        titles: vi.fn(async () => ({wearable: [], tracks: []})),
        fronts: vi.fn(async () => ({playsFor: [], playsAgainst: []})),
        wearTitle: vi.fn(async () => undefined),
    } satisfies PlayerBackend
    const store = new AccountStore(backend, player, {token: vi.fn(), held: vi.fn(), identity: vi.fn(), heldIdentity: vi.fn(), invalidate: vi.fn()}, {navigate: vi.fn(), remember: vi.fn()})
    await store.load()
    await vi.waitFor(() => expect(store.state()).toMatchObject({username: profile.name}))

    function Panel() {
        const state = useSyncExternalStore(store.subscribe, store.state)
        return state.kind === "ready" ? <AccountPanel state={state} store={store} onDelete={vi.fn()}/> : null
    }

    render(<Panel/>)
    const user = userEvent.setup()
    await user.click(screen.getByRole("tab", {name: "Account"}))
    return {player, user}
}

describe("UsernameForm", () => {
    it("waits until the name may change again, and says when", async () => {
        await signedIn({accountId: "account-1", name: "Ana", color: NameColor.UNSPECIFIED, renamableAt: Date.now() + 10 * DAY_MS},
            async (name) => ({accountId: "account-1", name}))

        expect((screen.getByLabelText("Username") as HTMLInputElement).disabled).toBe(true)
        expect((screen.getByRole("button", {name: "Save"}) as HTMLButtonElement).disabled).toBe(true)
        expect(screen.getByText(/^You can change it on /)).toBeDefined()
    })

    it("is open again once that time has passed", async () => {
        await signedIn({accountId: "account-1", name: "Ana", color: NameColor.UNSPECIFIED, renamableAt: Date.now() - 1000},
            async (name) => ({accountId: "account-1", name}))

        expect((screen.getByLabelText("Username") as HTMLInputElement).disabled).toBe(false)
        expect(screen.queryByText(/^You can change it on /)).toBeNull()
    })

    it("says when a change refused as too soon is allowed, and waits for it", async () => {
        const {user} = await signedIn({accountId: "account-1", name: "Ana", color: NameColor.UNSPECIFIED}, async () => {
            throw new RenameTooSoonError(Date.now() + 5 * DAY_MS)
        })

        await user.clear(screen.getByLabelText("Username"))
        await user.type(screen.getByLabelText("Username"), "Bob")
        await user.click(screen.getByRole("button", {name: "Save"}))

        expect(await screen.findByText(/^You can change it on /)).toBeDefined()
        expect((screen.getByLabelText("Username") as HTMLInputElement).disabled).toBe(true)
        expect(screen.queryByRole("alert")).toBeNull()
    })
})
