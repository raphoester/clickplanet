// @vitest-environment jsdom
import {StrictMode} from "react"
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {Countries} from "../../domain/countries.ts"
import {standingOf} from "../../domain/standing.ts"
import {nameOpenedFlags} from "../../i18n/translator.ts"
import CountryPicker from "../CountryPicker.tsx"
import {useStanding} from "./standingStore.ts"
import OutModal, {CONFIRM_ARM_MS} from "./OutModal.tsx"

const france = Countries.get("fr")!
// Sold since it left the build: no ground of its own, and no seat in an arena opened before it.
const england = {code: "gb-eng"}

// Paris fell; Berlin and Rome stand.
const standing = standingOf(new Map([[1, "fr"], [2, "de"], [3, "it"]]), new Set(["fr"]))

beforeEach(() => {
    vi.useFakeTimers({toFake: ["setTimeout", "clearTimeout"]})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    useStanding.setState(useStanding.getInitialState(), true)
})

const dialog = () => screen.queryByRole("dialog")
// fireEvent, not userEvent: userEvent waits on the timers these tests fake.
const click = (target: Element) => act(async () => { fireEvent.click(target) })
const wait = (ms: number) => act(() => { vi.advanceTimersByTime(ms) })
const option = (name: string) => screen.getByRole("option", {name})
const button = (name: string) => screen.getByRole("button", {name})
const options = () => within(screen.getByRole("listbox", {name: "Country"})).getAllByRole("option").map((option) => option.textContent)

describe("the out window", () => {
    it("tells a player whose flag is out to pick a flag still in, from those alone, and pledges it once confirmed", async () => {
        useStanding.getState().follow(standing)
        const onPick = vi.fn(async () => {})
        render(<StrictMode><OutModal flag={france} onPick={onPick}/></StrictMode>)

        expect(screen.getByRole("dialog", {name: "France is out"})).toBeDefined()
        expect(screen.getByText("Pick a flag that is still in.")).toBeDefined()
        expect(options()).toEqual(["Germany", "Italy"])

        await click(option("Germany"))
        expect(screen.getByRole("dialog", {name: "Play for Germany?"})).toBeDefined()
        expect(onPick).not.toHaveBeenCalled()

        wait(CONFIRM_ARM_MS)
        await click(button("Confirm"))
        expect(onPick).toHaveBeenCalledWith(Countries.get("de"))
    })

    it("takes no confirm the first moment it shows, so a double click on a country pledges nothing", async () => {
        useStanding.getState().follow(standing)
        const onPick = vi.fn(async () => {})
        render(<OutModal flag={france} onPick={onPick}/>)

        await click(option("Germany"))
        await click(button("Confirm"))
        wait(CONFIRM_ARM_MS - 100)
        await click(button("Confirm"))
        expect(onPick).not.toHaveBeenCalled()
        expect((button("Confirm") as HTMLButtonElement).disabled).toBe(true)

        wait(100)
        expect((button("Confirm") as HTMLButtonElement).disabled).toBe(false)
        await click(button("Confirm"))
        expect(onPick).toHaveBeenCalledWith(Countries.get("de"))
    })

    it("goes back to the list, and waits again on the next pick, even of the same country", async () => {
        useStanding.getState().follow(standing)
        const onPick = vi.fn(async () => {})
        render(<OutModal flag={france} onPick={onPick}/>)

        await click(option("Germany"))
        wait(CONFIRM_ARM_MS)
        await click(button("Back"))
        expect(screen.getByRole("dialog", {name: "France is out"})).toBeDefined()

        await click(option("Germany"))
        await click(button("Confirm"))
        expect(onPick).not.toHaveBeenCalled()

        wait(CONFIRM_ARM_MS)
        await click(button("Confirm"))
        expect(onPick).toHaveBeenCalledWith(Countries.get("de"))
    })

    it("says the flag picked is kept under the rules' lock, and nothing with no lock", async () => {
        useStanding.getState().follow(standing)
        const {rerender} = render(<OutModal flag={france} allegiance={{countryId: "fr", locked: true}} onPick={vi.fn()}/>)
        await click(option("Italy"))
        expect(screen.getByText("You can't change your flag again unless this one is knocked out too.")).toBeDefined()

        rerender(<OutModal flag={france} allegiance={{countryId: "fr"}} onPick={vi.fn()}/>)
        expect(screen.queryByText(/change your flag again/)).toBeNull()
    })

    it("tells the player of a flag with no seat that it has no capital", () => {
        nameOpenedFlags((code) => code === england.code ? "England" : undefined)
        useStanding.getState().follow(standing)
        render(<OutModal flag={england} onPick={vi.fn()}/>)

        expect(screen.getByRole("dialog", {name: "England has no capital"})).toBeDefined()
        nameOpenedFlags(() => undefined)
    })

    it("shows nothing while the knock-out does not play, or for a flag still in", () => {
        const {rerender} = render(<OutModal flag={france} onPick={vi.fn()}/>)
        expect(dialog()).toBeNull()

        act(() => useStanding.getState().follow(standing))
        rerender(<OutModal flag={Countries.get("de")!} onPick={vi.fn()}/>)
        expect(dialog()).toBeNull()
    })

    it("closes to watch, and opens again on a click the flag could not make", async () => {
        useStanding.getState().follow(standing)
        render(<OutModal flag={france} onPick={vi.fn()}/>)

        await click(button("Close"))
        expect(dialog()).toBeNull()

        act(() => useStanding.getState().ask())
        expect(screen.getByRole("dialog", {name: "France is out"})).toBeDefined()
    })

    it("says so when the pick fails, and stays", async () => {
        useStanding.getState().follow(standing)
        vi.spyOn(console, "error").mockImplementation(() => {})
        render(<OutModal flag={france} onPick={vi.fn(async () => { throw new Error("refused") })}/>)

        await click(option("Italy"))
        wait(CONFIRM_ARM_MS)
        await click(button("Confirm"))

        expect(screen.getByRole("alert").textContent).toBe("Could not change your flag. Try again.")
        expect(screen.getByRole("dialog", {name: "Play for Italy?"})).toBeDefined()
    })
})

describe("the country picker while the knock-out plays", () => {
    it("lists only the flags still in, and is open to a player whose flag is out though the rules lock flags", () => {
        useStanding.getState().follow(standing)
        render(<CountryPicker country={france} setCountry={vi.fn()} allegiance={{countryId: "fr", locked: true}}/>)

        expect(options()).toEqual(["Germany", "Italy"])
    })

    it("stays locked for a player whose flag is still in", () => {
        useStanding.getState().follow(standing)
        render(<CountryPicker country={Countries.get("de")!} setCountry={vi.fn()} allegiance={{countryId: "de", locked: true}}/>)

        expect(screen.queryByRole("listbox", {name: "Country"})).toBeNull()
    })
})
