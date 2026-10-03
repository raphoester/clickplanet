// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import AddToCalendarButton from "./AddToCalendarButton.tsx"
import {SEASON_ZERO} from "../../backends/fakeSeasonBackend.ts"

const textOf = (file: Blob) => new Promise<string>((resolve) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as string)
    reader.readAsText(file)
})

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

describe("AddToCalendarButton", () => {
    it("downloads the finale as a calendar file, linking back to the game", async () => {
        const files: Blob[] = []
        URL.createObjectURL = vi.fn((file: Blob) => {
            files.push(file)
            return "blob:fake"
        })
        URL.revokeObjectURL = vi.fn()
        let downloaded = ""
        vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(function (this: HTMLAnchorElement) {
            downloaded = this.download
        })

        render(<AddToCalendarButton season={SEASON_ZERO}/>)
        await userEvent.setup().click(screen.getByRole("button", {name: "Add to calendar"}))

        expect(downloaded).toBe("clickplanet-season-0-finale.ics")
        expect(files).toHaveLength(1)
        expect(files[0].type).toBe("text/calendar")
        const text = await textOf(files[0])
        expect(text).toContain("\r\nDTSTART:20261031T210000Z\r\nDTEND:20261031T230000Z\r\n")
        expect(text).toContain(`\r\nURL:${window.location.origin}/play\r\n`)
    })
})
