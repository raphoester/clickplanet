// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import Menu from "./Menu.tsx"
import {Countries} from "../domain/countries.ts"
import type {LeaderboardEntry} from "../domain/leaderboard.ts"

const france = Countries.get("fr")!
const entry = (code: string, tiles: number) => ({country: Countries.get(code)!, tiles})

afterEach(cleanup)

function setup(leaderboard: LeaderboardEntry[] = []) {
    const setCountry = vi.fn()
    const view = render(
        <Menu country={france} setCountry={setCountry} leaderboard={leaderboard} tilesCount={1000}/>,
    )
    return {...view, setCountry, user: userEvent.setup()}
}

/** Minus the table's own header row. */
const leaderboardRows = () => screen.queryAllByRole("row").slice(1)
const countryPanel = () => screen.queryByRole("listbox", {name: "Country"})
const aboutPanel = () => screen.queryByText("The ultimate world war")
const button = (name: string | RegExp) => screen.getByRole("button", {name})
const actions = () => screen.queryByRole("button", {name: "About"})

describe("Menu", () => {
    it("starts on the leaderboard, with every panel closed", () => {
        setup([entry("fr", 500)])

        expect(leaderboardRows()).toHaveLength(1)
        expect(countryPanel()).toBeNull()
        expect(aboutPanel()).toBeNull()
    })

    it("expands a panel inside the card, with no backdrop over the globe", async () => {
        const {user, container} = setup()

        await user.click(button("About"))

        expect(aboutPanel()).not.toBeNull()
        expect(container.querySelector(".menu")!.contains(aboutPanel())).toBe(true)
        expect(container.querySelector(".modal")).toBeNull()
    })

    /**
     * Stacked instead, the card ran down the whole page and piled five buttons
     * at its bottom. A panel takes the place of what opened it.
     */
    it("replaces the leaderboard and the buttons with the panel", async () => {
        const {user} = setup([entry("fr", 500)])

        await user.click(button("About"))

        expect(leaderboardRows()).toHaveLength(0)
        expect(actions()).toBeNull()
    })

    it("walks back up to the menu on the panel's close button", async () => {
        const {user} = setup([entry("fr", 500)])

        await user.click(button("About"))
        await user.click(button("Back"))

        expect(aboutPanel()).toBeNull()
        expect(leaderboardRows()).toHaveLength(1)
        expect(actions()).not.toBeNull()
    })

    it("opens the country picker on the country button", async () => {
        const {user} = setup()

        await user.click(button(france.name))

        expect(countryPanel()).not.toBeNull()
        expect(aboutPanel()).toBeNull()
    })

    /** The card's identity should not blink out when a panel opens. */
    it("keeps the card header while a panel is open", async () => {
        const {user} = setup()
        await user.click(button("About"))

        expect(screen.getByRole("heading", {name: "ClickPlanet", level: 1})).toBeDefined()
    })

    /** The card already has a ClickPlanet header; the panel must not repeat it. */
    it("titles the about panel without repeating the card's header", async () => {
        const {user} = setup()
        await user.click(button("About"))

        expect(screen.getByRole("heading", {name: "About", level: 2})).toBeDefined()
        expect(screen.getAllByRole("heading", {name: "ClickPlanet"})).toHaveLength(1)
    })

    /**
     * Drilling in unmounts the button that was clicked, so focus lands on the
     * body unless something moves it.
     */
    it("moves focus into the panel when it opens", async () => {
        const {user} = setup()

        await user.click(button("About"))

        expect(document.activeElement).toBe(screen.getByRole("heading", {name: "About", level: 2}))
    })

    it("hands focus back to the button that opened the panel", async () => {
        const {user} = setup()

        await user.click(button("About"))
        await user.click(button("Back"))

        expect(document.activeElement).toBe(button("About"))
    })

    it("hands focus back to the country button, not the about one", async () => {
        const {user} = setup()

        await user.click(button(france.name))
        await user.click(button("Close"))

        expect(document.activeElement).toBe(button(france.name))
    })

    it("closes the panel on Escape", async () => {
        const {user} = setup([entry("fr", 500)])

        await user.click(button("About"))
        await user.keyboard("{Escape}")

        expect(aboutPanel()).toBeNull()
        expect(leaderboardRows()).toHaveLength(1)
    })

    /** It blocks nothing, so it must not claim to be a dialog. */
    it("exposes the panel as a labelled region, not a dialog", async () => {
        const {user, container} = setup()

        await user.click(button("About"))

        expect(screen.getByRole("region", {name: "About"})).toBeDefined()
        expect(container.querySelector("[aria-modal]")).toBeNull()
    })

    it("picks a country through the panel", async () => {
        const {user, setCountry} = setup()

        await user.click(button(france.name))
        await user.click(screen.getByRole("option", {name: "🇯🇵 Japan"}))

        expect(setCountry).toHaveBeenCalledWith(Countries.get("jp"))
    })
})
