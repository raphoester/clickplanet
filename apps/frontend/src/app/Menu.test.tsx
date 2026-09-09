// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import Menu from "./Menu.tsx"
import {Countries} from "../domain/countries.ts"
import type {LeaderboardEntry} from "../domain/leaderboard.ts"

const france = Countries.get("fr")!
const entry = (code: string, tiles: number) => ({country: Countries.get(code)!, tiles})

afterEach(cleanup)

function setup(leaderboard: LeaderboardEntry[] = [], country = france) {
    const setCountry = vi.fn()
    const view = render(
        <Menu country={country} setCountry={setCountry} leaderboard={leaderboard} tilesCount={1000}/>,
    )
    return {...view, setCountry, user: userEvent.setup()}
}

const leaderboardRows = () => screen.queryAllByRole("row").slice(1)
const countryPanel = () => screen.queryByRole("listbox", {name: "Country"})
const aboutDialog = () => screen.queryByRole("dialog", {name: "About ClickPlanet"})
const button = (name: string | RegExp) => screen.getByRole("button", {name})
const collapse = () => button("ClickPlanet menu")

describe("Menu", () => {
    it("starts on the leaderboard, with the picker and About closed", () => {
        setup([entry("fr", 500)])

        expect(leaderboardRows()).toHaveLength(1)
        expect(countryPanel()).toBeNull()
        expect(aboutDialog()).toBeNull()
    })

    describe("the collapse", () => {
        it("folds the card away, keeping the country and its rank on screen", async () => {
            const {user} = setup([entry("jp", 500), entry("fr", 250)])

            await user.click(collapse())

            expect(leaderboardRows()).toHaveLength(0)
            expect(screen.queryByRole("button", {name: "About"})).toBeNull()
            expect(screen.getByText("🇫🇷 France")).toBeDefined()
            expect(screen.getByText("#2")).toBeDefined()
        })

        it("unfolds it again", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(collapse())
            await user.click(collapse())

            expect(leaderboardRows()).toHaveLength(1)
        })

        it("says whether it is open, and what it controls", async () => {
            const {user, container} = setup()

            expect(collapse().getAttribute("aria-expanded")).toBe("true")
            const controlled = collapse().getAttribute("aria-controls")!
            expect(container.querySelector(`[id="${controlled}"]`)).not.toBeNull()

            await user.click(collapse())
            expect(collapse().getAttribute("aria-expanded")).toBe("false")
            expect(collapse().getAttribute("aria-controls")).toBeNull()
        })

        it("shows a dash rather than a rank when the country holds no tile", () => {
            setup([entry("jp", 500)])
            expect(screen.getByText("—")).toBeDefined()
        })

        it("opens unfolded when nothing says the viewport is compact", () => {
            setup([entry("fr", 500)])
            expect(leaderboardRows()).toHaveLength(1)
        })
    })

    describe("the country picker", () => {
        it("opens on the Change button, not on the country name", async () => {
            const {user} = setup()

            await user.click(button("Change"))

            expect(countryPanel()).not.toBeNull()
        })

        it("replaces the leaderboard and the buttons", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(button("Change"))

            expect(leaderboardRows()).toHaveLength(0)
            expect(screen.queryByRole("button", {name: "About"})).toBeNull()
        })

        it("walks back up on the back arrow, and offers no other way out", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(button("Change"))
            expect(screen.queryByRole("button", {name: /close/i})).toBeNull()

            await user.click(button("Back"))

            expect(countryPanel()).toBeNull()
            expect(leaderboardRows()).toHaveLength(1)
        })

        it("closes on Escape", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(button("Change"))
            await user.keyboard("{Escape}")

            expect(countryPanel()).toBeNull()
            expect(leaderboardRows()).toHaveLength(1)
        })

        it("keeps the card header, and does not repeat it", async () => {
            const {user} = setup()

            await user.click(button("Change"))

            expect(screen.getByRole("heading", {name: "ClickPlanet", level: 1})).toBeDefined()
            expect(screen.getByRole("heading", {name: "Change country", level: 2})).toBeDefined()
        })

        it("exposes the panel as a labelled region, not a dialog", async () => {
            const {user, container} = setup()

            await user.click(button("Change"))

            expect(screen.getByRole("region", {name: "Change country"})).toBeDefined()
            expect(container.querySelector("[aria-modal]")).toBeNull()
        })

        it("moves focus into the panel, and hands it back to Change", async () => {
            const {user} = setup()

            await user.click(button("Change"))
            expect(document.activeElement)
                .toBe(screen.getByRole("heading", {name: "Change country", level: 2}))

            await user.click(button("Back"))
            expect(document.activeElement).toBe(button("Change"))
        })

        it("picks a country and returns to the leaderboard", async () => {
            const {user, setCountry} = setup([entry("fr", 500)])

            await user.click(button("Change"))
            await user.click(screen.getByRole("option", {name: "🇯🇵 Japan"}))

            expect(setCountry).toHaveBeenCalledWith(Countries.get("jp"))
            expect(countryPanel()).toBeNull()
            expect(leaderboardRows()).toHaveLength(1)
        })
    })

    describe("About", () => {
        it("opens as a modal dialog over the page, not inside the card", async () => {
            const {user, container} = setup()

            await user.click(button("About"))

            const dialog = aboutDialog()!
            expect(dialog).not.toBeNull()
            expect(dialog.getAttribute("aria-modal")).toBe("true")
            expect(container.querySelector(".menu")!.contains(dialog)).toBe(false)
        })

        it("leaves the leaderboard standing behind it", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(button("About"))

            const table = screen.getByRole("region", {name: "Leaderboard"})
            expect(within(table).getByText("🇫🇷 France")).toBeDefined()
        })

        it("pins the coffee button outside the scrolling copy", async () => {
            const {user, container} = setup()

            await user.click(button("About"))

            const coffee = screen.getByRole("link", {name: "Buy me a coffee"})
            expect(container.querySelector(".modal-footer")!.contains(coffee)).toBe(true)
            expect(container.querySelector(".modal-body")!.contains(coffee)).toBe(false)
        })

        it("closes on the ×, and hands focus back to the About button", async () => {
            const {user} = setup()

            await user.click(button("About"))
            await user.click(button("Close"))

            expect(aboutDialog()).toBeNull()
            expect(document.activeElement).toBe(button("About"))
        })

        it("closes on Escape", async () => {
            const {user} = setup()

            await user.click(button("About"))
            await user.keyboard("{Escape}")

            expect(aboutDialog()).toBeNull()
        })
    })
})
