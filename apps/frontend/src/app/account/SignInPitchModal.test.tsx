// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {AccountBackend, Provider} from "../../backends/account.ts"
import {PlayerBackend} from "../../backends/player.ts"
import {AccountStore} from "./accountStore.ts"
import SignInPitchModal from "./SignInPitchModal.tsx"

afterEach(cleanup)

async function guest() {
    const backend = {
        signInOptions: vi.fn(async (): Promise<Provider[]> => ["google", "discord"]),
        me: vi.fn(async () => ({linked: []})),
        startSignIn: vi.fn(async () => "https://discord.example/authorize"),
        completeSignIn: vi.fn(async () => undefined),
        signOut: vi.fn(async () => undefined),
        signOutEverywhere: vi.fn(async () => undefined),
        deleteAccount: vi.fn(async () => undefined),
    } satisfies AccountBackend
    const player = {
        profile: vi.fn(async () => ({accountId: "account-1", name: ""})),
        setName: vi.fn(async (name: string) => ({accountId: "account-1", name})),
    } satisfies PlayerBackend
    const navigate = vi.fn()
    const store = new AccountStore(backend, player, {token: vi.fn(), held: vi.fn(), invalidate: vi.fn()}, {navigate, remember: vi.fn()})

    await store.load()
    const state = store.state()
    if (state.kind !== "ready") throw new Error(`the store is ${state.kind}`)

    return {store, state, backend, navigate}
}

describe("SignInPitchModal", () => {
    it("says what signing in is worth, and that nobody has to", async () => {
        const {store, state} = await guest()
        render(<SignInPitchModal state={state} store={store} multiplier={2} onClose={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "Click 2× faster"})).toBeDefined()
        expect(screen.getByText(/your clicks refill 2× as fast/)).toBeDefined()
        expect(screen.getByText("You do not need an account to play.")).toBeDefined()
    })

    it("signs in with the provider pressed", async () => {
        const {store, state, backend, navigate} = await guest()
        render(<SignInPitchModal state={state} store={store} multiplier={2} onClose={vi.fn()}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Sign in with Discord"}))

        expect(backend.startSignIn).toHaveBeenCalledWith("discord", "signIn")
        expect(navigate).toHaveBeenCalledWith("https://discord.example/authorize")
    })
})
