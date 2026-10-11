// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {MAX_NOTE_LENGTH, SupportNoSessionError, SupportRateLimitedError, SupportRejectedError} from "../../backends/support.ts"
import NoteModal from "./NoteModal.tsx"

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

function setup(onSend: (text: string) => Promise<void> = vi.fn(async () => undefined)) {
    const onClose = vi.fn()
    render(<NoteModal title="Suggest a feature"
                      label="Your idea"
                      placeholder="What should the game have?"
                      onSend={onSend}
                      onClose={onClose}/>)
    return {onSend, onClose, user: userEvent.setup()}
}

const box = () => screen.getByRole("textbox", {name: "Your idea"}) as HTMLTextAreaElement
const send = () => screen.getByRole("button", {name: "Send"}) as HTMLButtonElement

describe("NoteModal", () => {
    it("opens on its text box, with nothing to send yet", () => {
        setup()

        expect(screen.getByRole("dialog", {name: "Suggest a feature"})).not.toBeNull()
        expect(document.activeElement).toBe(box())
        expect(send().disabled).toBe(true)
    })

    it("sends the text trimmed, lines kept, and says it was sent", async () => {
        const {onSend, onClose, user} = setup()

        await user.type(box(), "  a night mode{Enter}for the globe  ")
        await user.click(send())

        expect(onSend).toHaveBeenCalledWith("a night mode\nfor the globe")
        expect(screen.getByRole("status").textContent).toBe("Sent. Thank you!")
        expect(screen.queryByRole("textbox")).toBeNull()

        await user.click(screen.getAllByRole("button", {name: "Close"}).at(-1)!)
        expect(onClose).toHaveBeenCalledTimes(1)
    })

    it("sends nothing while one send is in flight", async () => {
        let land = () => {}
        const onSend = vi.fn(() => new Promise<void>((resolve) => {
            land = resolve
        }))
        const {user} = setup(onSend)

        await user.type(box(), "a night mode")
        await user.click(send())

        expect(send().disabled).toBe(true)
        expect(box().readOnly).toBe(true)
        await user.click(send())
        expect(onSend).toHaveBeenCalledTimes(1)

        land()
        expect((await screen.findByRole("status")).textContent).toBe("Sent. Thank you!")
    })

    it.each([
        [new SupportRateLimitedError(), "You sent too many. Try again in a few minutes."],
        [new SupportRejectedError(), "That was refused."],
        [new SupportNoSessionError(), "Could not start a session. Try again in a moment."],
    ])("says why %s refused it, and keeps the text", async (refusal, notice) => {
        const {user} = setup(vi.fn(async () => {
            throw refusal
        }))

        await user.type(box(), "a night mode")
        await user.click(send())

        expect(screen.getByRole("alert").textContent).toBe(notice)
        expect(box().value).toBe("a night mode")
        expect(send().disabled).toBe(false)
    })

    it("says a failure it does not know could not be sent, and logs it", async () => {
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})
        const {user} = setup(vi.fn(async () => {
            throw new Error("the connection dropped")
        }))

        await user.type(box(), "a night mode")
        await user.click(send())

        expect(screen.getByRole("alert").textContent).toBe("It could not be sent. Try again.")
        expect(logged).toHaveBeenCalledTimes(1)
    })

    it("counts down near the limit, and refuses a text past it", async () => {
        const {user} = setup()

        await user.click(box())
        await user.paste("a".repeat(MAX_NOTE_LENGTH - 5))
        expect(screen.getByText("5")).not.toBeNull()
        expect(send().disabled).toBe(false)

        await user.paste("🌍".repeat(6))
        expect(screen.getByText("-1")).not.toBeNull()
        expect(send().disabled).toBe(true)
    })

    it("does not close on a click beside it once something is typed", async () => {
        const {onClose, user} = setup()
        const backdrop = document.querySelector(".modal")!

        await user.click(backdrop)
        expect(onClose).toHaveBeenCalledTimes(1)

        await user.type(box(), "a night mode")
        await user.click(backdrop)
        expect(onClose).toHaveBeenCalledTimes(1)
    })
})
