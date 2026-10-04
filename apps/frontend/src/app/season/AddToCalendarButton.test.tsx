// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import AddToCalendarButton from "./AddToCalendarButton.tsx"
import {SEASON_ZERO} from "../../backends/fakeSeasonBackend.ts"

const season = {...SEASON_ZERO, finaleFile: "https://api.clickplanet.lol/seasons/0/finale.ics"}
const button = () => screen.getByRole("button", {name: "Add to calendar"})
const links = () => screen.queryAllByRole("link")

afterEach(cleanup)

describe("AddToCalendarButton", () => {
    it("opens a choice of calendars, and closes it on a second press", async () => {
        const user = userEvent.setup()
        render(<AddToCalendarButton season={season}/>)
        expect(button().getAttribute("aria-expanded")).toBe("false")
        expect(links()).toHaveLength(0)

        await user.click(button())
        expect(button().getAttribute("aria-expanded")).toBe("true")
        expect(links().map(link => link.textContent)).toEqual(["Google Calendar", "Apple Calendar", "Outlook"])

        await user.click(button())
        expect(links()).toHaveLength(0)
    })

    it("opens a web calendar in a new tab and the file in this one", async () => {
        render(<AddToCalendarButton season={season}/>)
        await userEvent.setup().click(button())

        const google = screen.getByRole("link", {name: "Google Calendar"})
        expect(new URL(google.getAttribute("href")!).origin).toBe("https://calendar.google.com")
        expect(new URL(google.getAttribute("href")!).searchParams.get("details")).toBe(`${window.location.origin}/play`)
        expect(google.getAttribute("target")).toBe("_blank")
        expect(google.getAttribute("rel")).toBe("noopener noreferrer")

        const apple = screen.getByRole("link", {name: "Apple Calendar"})
        expect(apple.getAttribute("href")).toBe("https://api.clickplanet.lol/seasons/0/finale.ics")
        expect(apple.getAttribute("target")).toBeNull()
        expect(apple.hasAttribute("download")).toBe(false)
    })

    it("closes on a pick", async () => {
        const user = userEvent.setup()
        render(<AddToCalendarButton season={season}/>)
        await user.click(button())

        const apple = screen.getByRole("link", {name: "Apple Calendar"})
        apple.addEventListener("click", (event) => event.preventDefault())
        await user.click(apple)

        expect(links()).toHaveLength(0)
    })

    it("closes on Escape and on a press elsewhere", async () => {
        const user = userEvent.setup()
        render(<>
            <AddToCalendarButton season={season}/>
            <p>elsewhere</p>
        </>)

        await user.click(button())
        await user.keyboard("{Escape}")
        expect(links()).toHaveLength(0)

        await user.click(button())
        await user.click(screen.getByText("elsewhere"))
        expect(links()).toHaveLength(0)
    })

    it("offers no Apple Calendar without a file", async () => {
        render(<AddToCalendarButton season={SEASON_ZERO}/>)
        await userEvent.setup().click(button())

        expect(links().map(link => link.textContent)).toEqual(["Google Calendar", "Outlook"])
    })
})
