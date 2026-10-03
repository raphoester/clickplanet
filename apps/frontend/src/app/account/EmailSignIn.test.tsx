// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {useSyncExternalStore} from "react"
import {AccountBackend, AuthError, Provider} from "../../backends/account.ts"
import {NameColor, PlayerBackend} from "../../backends/player.ts"
import {AccountStore} from "./accountStore.ts"
import AccountPanel from "./AccountPanel.tsx"

afterEach(cleanup)

async function guest(offered: Provider[] = ["google", "email"]) {
    const backend = {
        signInOptions: vi.fn(async (): Promise<Provider[]> => offered),
        me: vi.fn(async (): Promise<{linked: Provider[]}> => ({linked: []})),
        startSignIn: vi.fn(async () => "https://google.example/authorize"),
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
    const store = new AccountStore(backend, player, {token: vi.fn(), held: vi.fn(), invalidate: vi.fn()}, {navigate: vi.fn(), remember: vi.fn()})
    await store.load()

    function Panel() {
        const state = useSyncExternalStore(store.subscribe, store.state)
        return state.kind === "ready" ? <AccountPanel state={state} store={store} onDelete={vi.fn()}/> : null
    }

    render(<Panel/>)
    return {backend, user: userEvent.setup()}
}

describe("EmailSignIn", () => {
    it("is offered beside the buttons, and not as a button", async () => {
        await guest()

        expect(screen.getByRole("button", {name: "Sign in with Google"})).toBeDefined()
        expect(screen.getByLabelText("Or with your email")).toBeDefined()
        expect(screen.queryByRole("button", {name: /Sign in with email/i})).toBeNull()
    })

    it("drops the \"or\" when it is the only way in", async () => {
        await guest(["email"])

        expect(screen.getByLabelText("With your email")).toBeDefined()
    })

    it("is not offered when the server does not offer it", async () => {
        await guest(["google"])

        expect(screen.queryByLabelText("Or with your email")).toBeNull()
    })

    it("sends a code only to something that looks like an address", async () => {
        const {backend, user} = await guest()
        const send = screen.getByRole("button", {name: "Send code"}) as HTMLButtonElement

        await user.type(screen.getByLabelText("Or with your email"), "player")
        expect(send.disabled).toBe(true)

        await user.type(screen.getByLabelText("Or with your email"), "@example.com")
        await user.click(send)

        expect(backend.startEmailSignIn).toHaveBeenCalledWith("player@example.com", "signIn")
        expect(await screen.findByText("player@example.com")).toBeDefined()
    })

    it("signs in with the six digits typed, whatever else was pasted", async () => {
        const {backend, user} = await guest()
        await user.type(screen.getByLabelText("Or with your email"), "player@example.com")
        await user.click(screen.getByRole("button", {name: "Send code"}))

        await user.type(await screen.findByLabelText("Code"), "123 456")
        await user.click(screen.getByRole("button", {name: "Sign in"}))

        expect(backend.completeEmailSignIn).toHaveBeenCalledWith("123456")
    })

    it("says a wrong code and keeps the code step", async () => {
        const {backend, user} = await guest()
        backend.completeEmailSignIn.mockImplementation(async () => {
            throw new AuthError("wrongCode")
        })
        await user.type(screen.getByLabelText("Or with your email"), "player@example.com")
        await user.click(screen.getByRole("button", {name: "Send code"}))

        await user.type(await screen.findByLabelText("Code"), "999999")
        await user.click(screen.getByRole("button", {name: "Sign in"}))

        expect((await screen.findByRole("alert")).textContent).toMatch(/This code is not correct/)
        expect(screen.getByLabelText("Code")).toBeDefined()
    })

    it("goes back to the address", async () => {
        const {user} = await guest()
        await user.type(screen.getByLabelText("Or with your email"), "player@example.com")
        await user.click(screen.getByRole("button", {name: "Send code"}))

        await user.click(await screen.findByRole("button", {name: "Use another address"}))

        expect(screen.getByLabelText("Or with your email")).toBeDefined()
    })
})
