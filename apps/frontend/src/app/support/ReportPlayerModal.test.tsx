// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {PlayerReport, SupportReportedAlreadyError, SupportUnknownPlayerError} from "../../backends/support.ts"
import ReportPlayerModal from "./ReportPlayerModal.tsx"

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

function setup(reportPlayer: (report: PlayerReport) => Promise<void> = vi.fn(async () => undefined), messageId?: string) {
    const onClose = vi.fn()
    render(<ReportPlayerModal name="guest_a1b2c3" messageId={messageId} reporter={{reportPlayer}} onClose={onClose}/>)
    return {reportPlayer, onClose, user: userEvent.setup()}
}

const reason = (name: string) => screen.getByRole("radio", {name}) as HTMLButtonElement
const send = () => screen.getByRole("button", {name: "Send"}) as HTMLButtonElement
const details = () => screen.getByRole("textbox", {name: "Details"}) as HTMLTextAreaElement

describe("ReportPlayerModal", () => {
    it("names the player, asks why first, and sends nothing before a reason is picked", () => {
        setup()

        expect(screen.getByRole("dialog", {name: "Report guest_a1b2c3"})).not.toBeNull()
        expect(screen.getAllByRole("radio").map((option) => option.textContent))
            .toEqual(["Cheating or a bot", "Abuse in the chat", "Offensive name", "Something else"])
        expect(document.activeElement).toBe(reason("Cheating or a bot"))
        expect(send().disabled).toBe(true)
    })

    it("sends the reason alone: the details are optional", async () => {
        const {reportPlayer, user} = setup()

        await user.click(reason("Cheating or a bot"))
        expect(reason("Cheating or a bot").getAttribute("aria-checked")).toBe("true")
        await user.click(send())

        expect(reportPlayer).toHaveBeenCalledWith({name: "guest_a1b2c3", reason: "cheating", text: "", messageId: undefined})
        expect(screen.getByRole("status").textContent).toBe("Sent. Thank you!")
    })

    it("sends the details and the message the card was opened from", async () => {
        const {reportPlayer, user} = setup(undefined, "message-1")

        await user.click(reason("Offensive name"))
        await user.click(reason("Abuse in the chat"))
        await user.type(details(), "  it insults everyone  ")
        await user.click(send())

        expect(reportPlayer).toHaveBeenCalledWith(
            {name: "guest_a1b2c3", reason: "chat-abuse", text: "it insults everyone", messageId: "message-1"})
    })

    it.each([
        [new SupportUnknownPlayerError(), "No player holds this name now."],
        [new SupportReportedAlreadyError(), "You already reported this player today."],
    ])("says why %s refused it, and keeps the form", async (refusal, notice) => {
        const {user} = setup(vi.fn(async () => {
            throw refusal
        }))

        await user.click(reason("Something else"))
        await user.type(details(), "it never sleeps")
        await user.click(send())

        expect(screen.getByRole("alert").textContent).toBe(notice)
        expect(details().value).toBe("it never sleeps")
        expect(reason("Something else").getAttribute("aria-checked")).toBe("true")
    })
})
