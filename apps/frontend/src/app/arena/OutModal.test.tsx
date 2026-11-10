// @vitest-environment jsdom
import {StrictMode} from "react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {Countries} from "../../domain/countries.ts"
import {arenaFlagsOf} from "../../domain/standing.ts"
import CountryPicker from "../CountryPicker.tsx"
import {useArena} from "./arenaStore.ts"
import OutModal from "./OutModal.tsx"

const france = Countries.get("fr")!
const england = Countries.get("gb-eng")!

// Paris fell; Berlin and Rome stand.
const arena = arenaFlagsOf(new Map([[1, "fr"], [2, "de"], [3, "it"]]), new Set(["fr"]))

afterEach(() => {
    cleanup()
    useArena.setState(useArena.getInitialState(), true)
})

const dialog = () => screen.queryByRole("dialog")
const options = () => within(screen.getByRole("listbox", {name: "Country"})).getAllByRole("option").map((option) => option.textContent)

describe("the out window", () => {
    it("tells a player whose flag is out to pick a flag still in, from those alone", async () => {
        useArena.getState().follow(arena)
        const onPick = vi.fn(async () => {})
        render(<StrictMode><OutModal flag={france} onPick={onPick}/></StrictMode>)

        expect(screen.getByRole("dialog", {name: "France is out"})).toBeDefined()
        expect(screen.getByText("Pick a flag that is still in.")).toBeDefined()
        expect(options()).toEqual(["Germany", "Italy"])

        await userEvent.click(screen.getByRole("option", {name: "Germany"}))
        expect(onPick).toHaveBeenCalledWith(Countries.get("de"))
    })

    it("tells a region flag's player that it has no capital", () => {
        useArena.getState().follow(arena)
        render(<OutModal flag={england} onPick={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "England has no capital"})).toBeDefined()
    })

    it("shows nothing outside an arena or for a flag still in", () => {
        const {rerender} = render(<OutModal flag={france} onPick={vi.fn()}/>)
        expect(dialog()).toBeNull()

        act(() => useArena.getState().follow(arena))
        rerender(<OutModal flag={Countries.get("de")!} onPick={vi.fn()}/>)
        expect(dialog()).toBeNull()
    })

    it("closes to watch, and opens again on a click the flag could not make", async () => {
        useArena.getState().follow(arena)
        render(<OutModal flag={france} onPick={vi.fn()}/>)

        await userEvent.click(screen.getByRole("button", {name: "Close"}))
        expect(dialog()).toBeNull()

        act(() => useArena.getState().ask())
        expect(screen.getByRole("dialog", {name: "France is out"})).toBeDefined()
    })

    it("says so when the pick fails, and stays", async () => {
        useArena.getState().follow(arena)
        vi.spyOn(console, "error").mockImplementation(() => {})
        render(<OutModal flag={france} onPick={vi.fn(async () => { throw new Error("refused") })}/>)

        await userEvent.click(screen.getByRole("option", {name: "Italy"}))

        expect(screen.getByRole("alert").textContent).toBe("Could not change your flag. Try again.")
        expect(dialog()).not.toBeNull()
    })
})

describe("the country picker in an arena", () => {
    it("lists only the flags still in, and is open to a player whose flag is out though the rules lock flags", () => {
        useArena.getState().follow(arena)
        render(<CountryPicker country={france} setCountry={vi.fn()} allegiance={{countryId: "fr", locked: true}}/>)

        expect(options()).toEqual(["Germany", "Italy"])
    })

    it("stays locked for a player whose flag is still in", () => {
        useArena.getState().follow(arena)
        render(<CountryPicker country={Countries.get("de")!} setCountry={vi.fn()} allegiance={{countryId: "de", locked: true}}/>)

        expect(screen.queryByRole("listbox", {name: "Country"})).toBeNull()
    })
})
