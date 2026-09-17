// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {StrictMode} from "react"
import {AuthError, AuthFailure} from "../../backends/account.ts"
import SignInCallback from "./SignInCallback.tsx"

afterEach(cleanup)

const code = {kind: "code" as const, code: "abc", state: "xyz"}
const refusing = (failure: AuthFailure) => async (): Promise<void> => {
    throw new AuthError(failure)
}

describe("SignInCallback", () => {
    // StrictMode runs effects twice, and a second trade of the same code fails.
    it("trades the code once, then goes back to the game", async () => {
        const complete = vi.fn(async () => undefined)
        const onDone = vi.fn()

        render(<StrictMode><SignInCallback callback={code} complete={complete} onDone={onDone}/></StrictMode>)

        await vi.waitFor(() => expect(onDone).toHaveBeenCalledTimes(1))
        expect(complete).toHaveBeenCalledTimes(1)
        expect(complete).toHaveBeenCalledWith("abc", "xyz")
    })

    it("sends the same code again after a spent budget", async () => {
        const complete = vi.fn(refusing("tooManyTries"))
        const onDone = vi.fn()
        render(<SignInCallback callback={code} complete={complete} onDone={onDone}/>)

        expect(await screen.findByRole("alert")).toBeDefined()
        complete.mockImplementation(async () => undefined)
        await userEvent.setup().click(screen.getByRole("button", {name: "Try again"}))

        await vi.waitFor(() => expect(onDone).toHaveBeenCalledTimes(1))
        expect(complete).toHaveBeenCalledTimes(2)
    })

    it("goes back to the provider when the code is spent", async () => {
        const startAgain = vi.fn(async () => undefined)
        render(<SignInCallback callback={code} complete={refusing("startAgain")} startAgain={startAgain} onDone={vi.fn()}/>)

        await screen.findByRole("alert")
        await userEvent.setup().click(screen.getByRole("button", {name: "Try again"}))

        expect(startAgain).toHaveBeenCalledTimes(1)
    })

    it("offers only the way back when no provider is remembered", async () => {
        render(<SignInCallback callback={code} complete={refusing("startAgain")} onDone={vi.fn()}/>)

        await screen.findByRole("alert")

        expect(screen.queryByRole("button", {name: "Try again"})).toBeNull()
        expect(screen.getByRole("button", {name: "Back to the game"})).toBeDefined()
    })

    // Nothing changed on the server: the player goes back on the account they were on.
    it("explains a refused link and goes back to the game", async () => {
        const onDone = vi.fn()
        render(<SignInCallback callback={code} provider="google" complete={refusing("linkedElsewhere")}
                               startAgain={vi.fn(async () => undefined)} onDone={onDone}/>)

        expect((await screen.findByRole("alert")).textContent).toBe(
            "This Google account is already used by another ClickPlanet account. "
            + "To move it here: sign in with it, delete that account, then link it here.",
        )
        expect(screen.getByRole("heading", {name: "Not linked"})).toBeDefined()
        expect(screen.queryByRole("button", {name: "Try again"})).toBeNull()

        await userEvent.setup().click(screen.getByRole("button", {name: "Back to the game"}))
        expect(onDone).toHaveBeenCalledTimes(1)
    })

    it("trades nothing when the player said no", async () => {
        const complete = vi.fn(async () => undefined)
        const onDone = vi.fn()
        render(<SignInCallback callback={{kind: "declined"}} complete={complete} onDone={onDone}/>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Back to the game"}))

        expect(complete).not.toHaveBeenCalled()
        expect(onDone).toHaveBeenCalledTimes(1)
    })
})
