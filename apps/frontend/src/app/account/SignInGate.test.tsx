// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import {StrictMode} from "react"
import {FakeAccountBackend} from "../../backends/fakeAccountBackend.ts"
import {NoSession} from "../../backends/session.ts"
import {AccountStore} from "./accountStore.ts"
import {rememberSignIn} from "./rememberedSignIn.ts"
import SignInGate from "./SignInGate.tsx"

const code = {kind: "code" as const, code: "abc", state: "xyz"}

function setup() {
    const accountBackend = new FakeAccountBackend()
    const account = new AccountStore(accountBackend, accountBackend, accountBackend, new NoSession(), {navigate: vi.fn(), remember: rememberSignIn, claimFlag: vi.fn()})
    const completeSignIn = vi.spyOn(accountBackend, "completeSignIn")

    render(<StrictMode>
        <SignInGate callback={code} account={account}>
            <p>the game</p>
        </SignInGate>
    </StrictMode>)

    return {completeSignIn, account}
}

beforeEach(() => window.sessionStorage.clear())
afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

describe("SignInGate", () => {
    it("trades the code once, as a sign-in, and swaps the game in", async () => {
        rememberSignIn("discord", "signIn")

        const {completeSignIn} = setup()

        expect(await screen.findByText("the game")).toBeDefined()
        expect(completeSignIn).toHaveBeenCalledTimes(1)
        expect(completeSignIn).toHaveBeenCalledWith("abc", "xyz")
    })

    it("tells the account the sign-in was Discord's, which joins the server a moment later", async () => {
        rememberSignIn("discord", "link")

        const {account} = setup()

        expect(await screen.findByText("the game")).toBeDefined()
        expect(account.state()).toMatchObject({kind: "ready", joiningDiscord: true})
    })

    it("tells no other provider's sign-in as Discord's", async () => {
        rememberSignIn("google", "signIn")

        const {account} = setup()

        expect(await screen.findByText("the game")).toBeDefined()
        expect(account.state()).not.toHaveProperty("joiningDiscord")
    })
})
