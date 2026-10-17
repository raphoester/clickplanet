// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {ChatMessage} from "../../backends/chat.ts"
import {NameColor} from "../../backends/player.ts"
import {PlayerReport, SupportReportedAlreadyError} from "../../backends/support.ts"
import ReportMessageModal from "./ReportMessageModal.tsx"

afterEach(cleanup)

const insult: ChatMessage = {
    id: "message-1", sentAt: Date.UTC(2026, 9, 17, 12), authorAccountId: "account-bo", authorName: "guest_b0b0b0",
    authorAdmin: false, authorColor: NameColor.UNSPECIFIED, authorStreak: 0, countryCode: "de",
    text: "you are all idiots", reactions: [], reactionsVersion: 0,
}

function setup(reportPlayer: (report: PlayerReport) => Promise<void> = vi.fn(async () => undefined)) {
    const onSent = vi.fn()
    render(<ReportMessageModal message={insult} reporter={{reportPlayer}} onSent={onSent} onClose={() => {}}/>)
    return {reportPlayer, onSent, user: userEvent.setup()}
}

describe("ReportMessageModal", () => {
    it("shows who wrote what, and asks for no reason", () => {
        setup()

        expect(screen.getByRole("dialog", {name: "Report a message"})).not.toBeNull()
        expect(screen.getByText("guest_b0b0b0")).toBeDefined()
        expect(screen.getByText("you are all idiots")).toBeDefined()
        expect(screen.queryAllByRole("radio")).toHaveLength(0)
        expect(document.activeElement).toBe(screen.getByRole("textbox", {name: "Details"}))
    })

    it("sends the message as abuse in the chat, the details being optional", async () => {
        const {reportPlayer, onSent, user} = setup()

        await user.click(screen.getByRole("button", {name: "Send"}))

        expect(reportPlayer).toHaveBeenCalledWith({accountId: "account-bo", reason: "chat-abuse", text: "", messageId: "message-1"})
        expect(onSent).toHaveBeenCalledWith(insult)
        expect(screen.getByRole("status").textContent).toBe("Sent. Thank you!")
    })

    it("says why a report was refused, keeps the details, and marks nothing", async () => {
        const {onSent, user} = setup(vi.fn(async () => {
            throw new SupportReportedAlreadyError()
        }))

        await user.type(screen.getByRole("textbox", {name: "Details"}), "every day")
        await user.click(screen.getByRole("button", {name: "Send"}))

        expect(screen.getByRole("alert").textContent).toBe("You already reported this player today.")
        expect(screen.getByRole("textbox", {name: "Details"})).toHaveProperty("value", "every day")
        expect(onSent).not.toHaveBeenCalled()
    })
})
