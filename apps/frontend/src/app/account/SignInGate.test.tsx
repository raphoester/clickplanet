// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import {StrictMode} from "react"
import {FakeAccountBackend} from "../../backends/fakeAccountBackend.ts"
import {FakeAlertsBackend} from "../../backends/fakeAlertsBackend.ts"
import {NoSession} from "../../backends/session.ts"
import {DiscordAlerts} from "../alerts/discordAlerts.ts"
import {AccountStore} from "./accountStore.ts"
import {rememberDiscordLink, rememberSignIn} from "./rememberedSignIn.ts"
import SignInGate from "./SignInGate.tsx"

const code = {kind: "code" as const, code: "abc", state: "xyz"}

function setup() {
    const accountBackend = new FakeAccountBackend()
    const alertsBackend = new FakeAlertsBackend()
    const account = new AccountStore(accountBackend, accountBackend, new NoSession(), {navigate: vi.fn(), remember: rememberSignIn})
    const alerts = new DiscordAlerts(alertsBackend, {navigate: vi.fn(), remember: rememberDiscordLink})
    const completeSignIn = vi.spyOn(accountBackend, "completeSignIn")
    const completeDiscordLink = vi.spyOn(alertsBackend, "completeDiscordLink")

    render(<StrictMode>
        <SignInGate callback={code} account={account} alerts={alerts}>
            <p>the game</p>
        </SignInGate>
    </StrictMode>)

    return {completeSignIn, completeDiscordLink}
}

beforeEach(() => window.sessionStorage.clear())
afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

describe("SignInGate", () => {
    it("trades the code of a Discord link trip with the alerts, once", async () => {
        rememberDiscordLink()

        const {completeSignIn, completeDiscordLink} = setup()

        expect(await screen.findByRole("heading", {name: "You are in!"})).toBeDefined()
        expect(completeDiscordLink).toHaveBeenCalledTimes(1)
        expect(completeDiscordLink).toHaveBeenCalledWith("abc", "xyz")
        expect(completeSignIn).not.toHaveBeenCalled()
    })

    it("trades the code of a sign-in trip as a sign-in", async () => {
        rememberSignIn("google", "signIn")

        const {completeSignIn, completeDiscordLink} = setup()

        expect(await screen.findByText("the game")).toBeDefined()
        expect(completeSignIn).toHaveBeenCalledTimes(1)
        expect(completeDiscordLink).not.toHaveBeenCalled()
    })

    it("takes a sign-in started after a Discord link trip for a sign-in", async () => {
        rememberDiscordLink()
        rememberSignIn("discord", "link")

        const {completeSignIn, completeDiscordLink} = setup()

        await vi.waitFor(() => expect(completeSignIn).toHaveBeenCalledTimes(1))
        expect(completeDiscordLink).not.toHaveBeenCalled()
    })

    it("takes a Discord link trip started after a sign-in for a Discord link", async () => {
        rememberSignIn("google", "signIn")
        rememberDiscordLink()

        const {completeSignIn, completeDiscordLink} = setup()

        await vi.waitFor(() => expect(completeDiscordLink).toHaveBeenCalledTimes(1))
        expect(completeSignIn).not.toHaveBeenCalled()
    })
})
