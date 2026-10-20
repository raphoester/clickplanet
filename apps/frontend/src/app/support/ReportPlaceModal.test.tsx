// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {PlaceReport, SupportRateLimitedError} from "../../backends/support.ts"
import ReportPlaceModal from "./ReportPlaceModal.tsx"

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

function setup(reportPlace: (report: PlaceReport) => Promise<void> = vi.fn(async () => undefined)) {
    const onClose = vi.fn()
    render(<ReportPlaceModal tileId={4242} reporter={{reportPlace}} onClose={onClose}/>)
    return {reportPlace, onClose, user: userEvent.setup()}
}

const reason = (name: string) => screen.getByRole("radio", {name}) as HTMLButtonElement
const send = () => screen.getByRole("button", {name: "Send"}) as HTMLButtonElement
const details = () => screen.getByRole("textbox", {name: "Details"}) as HTMLTextAreaElement

describe("ReportPlaceModal", () => {
    it("asks what the drawing is first, and sends nothing before", () => {
        setup()

        expect(screen.getByRole("dialog", {name: "Report a drawing"})).not.toBeNull()
        expect(screen.getAllByRole("radio").map((option) => option.textContent))
            .toEqual(["A hateful symbol or words", "Something sexual", "Something else"])
        expect(send().disabled).toBe(true)
    })

    it("sends the tile picked, the reason and the details", async () => {
        const {reportPlace, user} = setup()

        await user.click(reason("A hateful symbol or words"))
        await user.type(details(), "  a swastika over Berlin  ")
        await user.click(send())

        expect(reportPlace).toHaveBeenCalledWith({tileId: 4242, reason: "hate", text: "a swastika over Berlin"})
        expect(screen.getByRole("status").textContent).toBe("Sent. Thank you!")
    })

    it("says why a report was refused and keeps it to send again", async () => {
        const {user} = setup(vi.fn(async () => {
            throw new SupportRateLimitedError()
        }))

        await user.click(reason("Something else"))
        await user.click(send())

        expect(await screen.findByText("You sent too many. Try again later.")).not.toBeNull()
        expect(send().disabled).toBe(false)
    })
})
