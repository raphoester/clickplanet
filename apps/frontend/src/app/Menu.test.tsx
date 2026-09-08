// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import Menu from "./Menu.tsx"
import {Countries} from "../domain/countries.ts"

const france = Countries.get("fr")!

afterEach(cleanup)

function setup() {
    const setCountry = vi.fn()
    const view = render(
        <Menu country={france} setCountry={setCountry} leaderboard={[]} tilesCount={1}/>,
    )
    return {...view, setCountry, user: userEvent.setup()}
}

const countryPanel = () => screen.queryByRole("listbox", {name: "Country"})
const aboutPanel = () => screen.queryByText("The ultimate world war")
const button = (name: string | RegExp) => screen.getByRole("button", {name})

describe("Menu", () => {
    it("starts with every panel closed", () => {
        setup()
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

    it("collapses the panel when its own button is pressed again", async () => {
        const {user} = setup()

        await user.click(button("About"))
        await user.click(button("About"))

        expect(aboutPanel()).toBeNull()
    })

    /** Two open at once would push the card past the bottom of the viewport. */
    it("closes the open panel when the other one is opened", async () => {
        const {user} = setup()

        await user.click(button("About"))
        await user.click(button(france.name))

        expect(aboutPanel()).toBeNull()
        expect(countryPanel()).not.toBeNull()
    })

    /** The card already has a ClickPlanet header; the panel must not repeat it. */
    it("titles the about panel without repeating the card's header", async () => {
        const {user} = setup()
        await user.click(button("About"))

        expect(screen.getByRole("heading", {name: "About", level: 2})).toBeDefined()
        // The card's own h1 stays; what must not appear is a second one.
        expect(screen.getAllByRole("heading", {name: "ClickPlanet"})).toHaveLength(1)
    })

    it("closes on the panel's own close button", async () => {
        const {user} = setup()

        await user.click(button("About"))
        await user.click(button("Back"))

        expect(aboutPanel()).toBeNull()
    })

    it("marks the open button as expanded", async () => {
        const {user} = setup()
        expect(button("About").getAttribute("aria-expanded")).toBe("false")

        await user.click(button("About"))

        expect(button("About").getAttribute("aria-expanded")).toBe("true")
    })
})