// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {AccountBackend, Provider} from "../../backends/account.ts"
import {NameColor, PlayerBackend} from "../../backends/player.ts"
import {AccountStore} from "./accountStore.ts"
import SignInNudgeModal from "./SignInNudgeModal.tsx"

afterEach(cleanup)

async function guest() {
    const backend = {
        signInOptions: vi.fn(async (): Promise<Provider[]> => ["google", "discord"]),
        me: vi.fn(async () => ({linked: []})),
        startSignIn: vi.fn(async () => "https://discord.example/authorize"),
        completeSignIn: vi.fn(async () => "signedIn" as const),
        startEmailSignIn: vi.fn(async () => undefined),
        completeEmailSignIn: vi.fn(async () => "signedIn" as const),
        completeCrazyGamesSignIn: vi.fn(async () => "signedIn" as const),
        signOut: vi.fn(async () => undefined),
        signOutEverywhere: vi.fn(async () => undefined),
        deleteAccount: vi.fn(async () => undefined),
    } satisfies AccountBackend
    const player = {
        profile: vi.fn(async () => ({accountId: "account-1", name: "", color: NameColor.UNSPECIFIED})),
        setName: vi.fn(async (name: string) => ({accountId: "account-1", name})),
        setColor: vi.fn(async (color: NameColor) => color),
        titles: vi.fn(async () => ({wearable: [], tracks: []})),
        fronts: vi.fn(async () => ({playsFor: [], playsAgainst: []})),
        wearTitle: vi.fn(async () => undefined),
    } satisfies PlayerBackend
    const navigate = vi.fn()
    const store = new AccountStore(backend, player, {token: vi.fn(), held: vi.fn(), identity: vi.fn(), heldIdentity: vi.fn(), invalidate: vi.fn()}, {navigate, remember: vi.fn()})

    await store.load()
    const state = store.state()
    if (state.kind !== "ready") throw new Error(`the store is ${state.kind}`)

    return {store, state, backend, navigate}
}

describe("SignInNudgeModal", () => {
    it("says what signing in gives a guest out of clicks", async () => {
        const {store, state} = await guest()
        render(<SignInNudgeModal state={state} store={store} multiplier={2} fullBank onClose={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "Out of clicks"})).toBeDefined()
        expect(screen.getAllByRole("listitem").map((item) => item.textContent)).toEqual([
            "A full bank now",
            "Clicks refill 2× faster",
            "Your land on every device",
        ])
    })

    it("promises no full bank the server does not give", async () => {
        const {store, state} = await guest()
        render(<SignInNudgeModal state={state} store={store} multiplier={2} fullBank={false} onClose={vi.fn()}/>)

        expect(screen.queryByText("A full bank now")).toBeNull()
    })

    it("signs in with the provider pressed", async () => {
        const {store, state, backend, navigate} = await guest()
        render(<SignInNudgeModal state={state} store={store} multiplier={2} fullBank onClose={vi.fn()}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Sign in with Google"}))

        expect(backend.startSignIn).toHaveBeenCalledWith("google", "signIn")
        expect(navigate).toHaveBeenCalledWith("https://discord.example/authorize")
    })
})
