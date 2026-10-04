// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {AccountBackend, Provider} from "../../backends/account.ts"
import {NameColor, PlayerBackend} from "../../backends/player.ts"
import {AccountStore} from "./accountStore.ts"
import SignInPitchModal from "./SignInPitchModal.tsx"

afterEach(cleanup)

async function guest() {
    const backend = {
        signInOptions: vi.fn(async (): Promise<Provider[]> => ["google", "discord"]),
        me: vi.fn(async () => ({linked: []})),
        startSignIn: vi.fn(async () => "https://discord.example/authorize"),
        completeSignIn: vi.fn(async () => undefined),
        startEmailSignIn: vi.fn(async () => undefined),
        completeEmailSignIn: vi.fn(async () => undefined),
        signOut: vi.fn(async () => undefined),
        signOutEverywhere: vi.fn(async () => undefined),
        deleteAccount: vi.fn(async () => undefined),
    } satisfies AccountBackend
    const player = {
        profile: vi.fn(async () => ({accountId: "account-1", name: "", color: NameColor.UNSPECIFIED})),
        setName: vi.fn(async (name: string) => ({accountId: "account-1", name})),
        setColor: vi.fn(async (color: NameColor) => color),
        titles: vi.fn(async () => ({wearable: [], tracks: []})),
        wearTitle: vi.fn(async () => undefined),
    } satisfies PlayerBackend
    const navigate = vi.fn()
    const store = new AccountStore(backend, player, {token: vi.fn(), held: vi.fn(), identity: vi.fn(), heldIdentity: vi.fn(), invalidate: vi.fn()}, {navigate, remember: vi.fn()})

    await store.load()
    const state = store.state()
    if (state.kind !== "ready") throw new Error(`the store is ${state.kind}`)

    return {store, state, backend, navigate}
}

describe("SignInPitchModal", () => {
    it("says what signing in is worth, and that nobody has to", async () => {
        const {store, state} = await guest()
        render(<SignInPitchModal state={state} store={store} multiplier={2} onClose={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "Sign in and stand out"})).toBeDefined()
        expect(screen.getByText(/refill 2× as fast/)).toBeDefined()
        expect(screen.getByText("It is free. You do not need an account to play.")).toBeDefined()
    })

    it("sells more than speed: a name, a color, a place on the board and a streak flame", async () => {
        const {store, state} = await guest()
        render(<SignInPitchModal state={state} store={store} multiplier={2} onClose={vi.fn()}/>)

        const perks = screen.getAllByRole("listitem").map((item) => item.querySelector("strong")?.textContent)
        expect(perks).toEqual([
            "Click 2× faster.",
            "Your name.",
            "Your color.",
            "Your place on the board.",
            "Your streak flame.",
        ])
    })

    it("signs in with the provider pressed", async () => {
        const {store, state, backend, navigate} = await guest()
        render(<SignInPitchModal state={state} store={store} multiplier={2} onClose={vi.fn()}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Sign in with Discord"}))

        expect(backend.startSignIn).toHaveBeenCalledWith("discord", "signIn")
        expect(navigate).toHaveBeenCalledWith("https://discord.example/authorize")
    })
})
