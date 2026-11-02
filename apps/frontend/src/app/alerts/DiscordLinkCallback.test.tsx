// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {StrictMode} from "react"
import {AlertsError, AlertsFailure} from "../../backends/alerts.ts"
import DiscordLinkCallback from "./DiscordLinkCallback.tsx"

afterEach(cleanup)

const code = {kind: "code" as const, code: "abc", state: "xyz"}
const SERVER = "https://discord.com/channels/1"
const refusing = (failure: AlertsFailure) => async (): Promise<string> => {
    throw new AlertsError(failure)
}

describe("DiscordLinkCallback", () => {
    it("trades the code once, then opens the server or goes back to the game", async () => {
        const complete = vi.fn(async () => SERVER)
        const onDone = vi.fn()

        render(<StrictMode>
            <DiscordLinkCallback callback={code} complete={complete} startAgain={vi.fn(async () => undefined)} onDone={onDone}/>
        </StrictMode>)

        expect(await screen.findByRole("heading", {name: "You are in!"})).toBeDefined()
        expect(complete).toHaveBeenCalledTimes(1)
        expect(complete).toHaveBeenCalledWith("abc", "xyz")
        const open = screen.getByRole("link", {name: "Open Discord"})
        expect(open.getAttribute("href")).toBe(SERVER)
        expect(open.getAttribute("target")).toBe("_blank")
        expect(open.getAttribute("rel")).toBe("noopener noreferrer")

        await userEvent.setup().click(screen.getByRole("button", {name: "Back to the game"}))
        expect(onDone).toHaveBeenCalledTimes(1)
    })

    it.each<[AlertsFailure, string]>([
        ["startAgain", "This link is too old, or it started in a different browser. Start again."],
        ["refused", "Discord did not accept it. Try again."],
        ["notSignedIn", "We could not check who you are. Try again."],
        ["tooManyTries", "Too many tries from your network. Wait one minute, then try again."],
        ["failed", "Something went wrong. Try again."],
    ])("says why a %s trade failed, and starts again from Discord", async (failure, message) => {
        const startAgain = vi.fn(async () => undefined)
        render(<DiscordLinkCallback callback={code} complete={refusing(failure)} startAgain={startAgain} onDone={vi.fn()}/>)

        expect((await screen.findByRole("alert")).textContent).toBe(message)
        await userEvent.setup().click(screen.getByRole("button", {name: "Try again"}))

        expect(startAgain).toHaveBeenCalledTimes(1)
    })

    it("offers only the way back when the pings are off", async () => {
        render(<DiscordLinkCallback callback={code} complete={refusing("off")} startAgain={vi.fn(async () => undefined)} onDone={vi.fn()}/>)

        expect((await screen.findByRole("alert")).textContent).toBe("Discord pings are not available now.")
        expect(screen.queryByRole("button", {name: "Try again"})).toBeNull()
        expect(screen.getByRole("button", {name: "Back to the game"})).toBeDefined()
    })

    it("says why a start again failed", async () => {
        render(<DiscordLinkCallback callback={code}
                                    complete={refusing("startAgain")}
                                    startAgain={async () => {
                                        throw new AlertsError("tooManyTries")
                                    }}
                                    onDone={vi.fn()}/>)

        await screen.findByRole("alert")
        await userEvent.setup().click(screen.getByRole("button", {name: "Try again"}))

        expect((await screen.findByRole("alert")).textContent).toBe("Too many tries from your network. Wait one minute, then try again.")
    })

    it("trades nothing when the player said no on Discord, and can start again", async () => {
        const complete = vi.fn(async () => SERVER)
        const startAgain = vi.fn(async () => undefined)
        render(<DiscordLinkCallback callback={{kind: "declined"}} complete={complete} startAgain={startAgain} onDone={vi.fn()}/>)

        expect(screen.getByRole("heading", {name: "You did not link Discord"})).toBeDefined()
        await userEvent.setup().click(screen.getByRole("button", {name: "Try again"}))

        expect(complete).not.toHaveBeenCalled()
        expect(startAgain).toHaveBeenCalledTimes(1)
    })
})
