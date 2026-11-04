// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {AuthError} from "../../backends/account.ts"
import DiscordButton from "./DiscordButton.tsx"

const SERVER = "https://discord.com/channels/1/2"

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

describe("DiscordButton", () => {
    it("joins by Discord's sign-in while the player is not in the server", async () => {
        const connect = vi.fn(async () => undefined)
        render(<DiscordButton offer={{connect}} className="button">Discord</DiscordButton>)

        const join = screen.getByRole("button", {name: "Join Discord"})
        await userEvent.setup().click(join)

        expect(connect).toHaveBeenCalledTimes(1)
        expect(join.hasAttribute("disabled")).toBe(true)
        expect(screen.queryByRole("link")).toBeNull()
    })

    it("says why the sign-in could not start, and takes another press", async () => {
        const connect = vi.fn(async () => {
            throw new AuthError("tooManyTries")
        })
        render(<DiscordButton offer={{connect}} className="button">Discord</DiscordButton>)

        await userEvent.setup().click(screen.getByRole("button", {name: "Join Discord"}))
        await userEvent.setup().click(await screen.findByRole("button", {name: "Wait a minute"}))

        expect(connect).toHaveBeenCalledTimes(2)
    })

    it("opens the server's page in a new tab once the player is in it", () => {
        render(<DiscordButton offer={{serverUrl: SERVER}} className="button">Discord</DiscordButton>)

        const open = screen.getByRole("link", {name: "Open Discord"})
        expect(open.getAttribute("href")).toBe(SERVER)
        expect(open.getAttribute("target")).toBe("_blank")
        expect(open.getAttribute("rel")).toBe("noopener noreferrer")
    })

    it("shows nothing when nothing is offered: an invite would join with no flag's role", () => {
        const none = render(<DiscordButton className="button">Discord</DiscordButton>)
        expect(none.container.innerHTML).toBe("")

        cleanup()
        const empty = render(<DiscordButton offer={{}} className="button">Discord</DiscordButton>)
        expect(empty.container.innerHTML).toBe("")
    })
})
