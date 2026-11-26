// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import CountryPicker from "./CountryPicker.tsx"
import {useFlagForm} from "./flags/flagFormStore.ts"
import {useFlags} from "./flags/flagsStore.ts"
import {useFlagSubmissions} from "./flags/submissionsStore.ts"
import type {ListedFlag} from "../domain/flags.ts"

const BAVARIA: ListedFlag = {code: "r-k3p9x2ma0b", name: "Bavaria", closed: false}
const SAVOY: ListedFlag = {code: "r-0a1b2c3d4e", name: "Savoy", closed: true}

afterEach(() => {
    cleanup()
    useFlags.getState().serve(undefined)
    useFlagForm.setState(useFlagForm.getInitialState(), true)
    useFlagSubmissions.setState(useFlagSubmissions.getInitialState(), true)
})

// Read off the options' ids and words: a query by role over 240 options takes seconds.
const optionOf = (container: HTMLElement, code: string) =>
    container.querySelector<HTMLElement>(`[role=option][id="country-option-${code}"]`)

async function serveList(listed: ListedFlag[]) {
    useFlags.getState().serve({listFlags: async () => listed})
    await vi.waitFor(() => expect(useFlags.getState().read).toBe(true))
}

describe("CountryPicker", () => {
    it("lists an opened flag by the name the list gives it, and picks it by its code", async () => {
        await serveList([BAVARIA])
        const setCountry = vi.fn()
        const {container} = render(<CountryPicker country={{code: "fr"}} setCountry={setCountry}/>)

        const option = optionOf(container, BAVARIA.code)!
        expect(option.textContent).toBe("Bavaria")
        await userEvent.click(option)

        expect(setCountry).toHaveBeenCalledWith({code: BAVARIA.code})
    })

    it("leaves a closed flag out, an opened one and one of the build alike", async () => {
        await serveList([{code: "de", name: "Germany", closed: true}, BAVARIA, SAVOY])
        const {container} = render(<CountryPicker country={{code: "fr"}} setCountry={vi.fn()}/>)

        expect(optionOf(container, SAVOY.code)).toBeNull()
        expect(optionOf(container, "de")).toBeNull()
        expect(optionOf(container, BAVARIA.code)).not.toBeNull()
        expect(optionOf(container, "fr")).not.toBeNull()
    })

    it("lists the flags of the build alone before the list is read", () => {
        const {container} = render(<CountryPicker country={{code: "fr"}} setCountry={vi.fn()}/>)

        expect(optionOf(container, BAVARIA.code)).toBeNull()
        expect(optionOf(container, "de")).not.toBeNull()
    })

    it("ends with the way to a region's flag, which a search that finds no country brings up, named as typed", async () => {
        useFlagSubmissions.setState({who: "player", submissions: [], read: true})
        render(<CountryPicker country={{code: "fr"}} setCountry={vi.fn()}/>)
        expect(screen.getByRole("button", {name: "Add your region's flag"})).not.toBeNull()

        await userEvent.type(screen.getByPlaceholderText("Search a country"), "Bavaria")
        expect(screen.queryByRole("button", {name: "Add your region's flag"})).toBeNull()
        await userEvent.click(screen.getByRole("button", {name: "Put “Bavaria” on the map"}))

        expect(useFlagForm.getState()).toMatchObject({shown: true, step: {kind: "make"}, draft: {name: "Bavaria"}})
    })

    it("keeps the way to a region's flag while the list is locked", () => {
        act(() => useFlagSubmissions.setState({who: "player"}))
        render(<CountryPicker country={{code: "fr"}} allegiance={{countryId: "fr", changeableAt: Date.now() + 60_000}} setCountry={vi.fn()}/>)

        expect(screen.queryByPlaceholderText("Search a country")).toBeNull()
        expect(screen.getByRole("button", {name: "Add your region's flag"})).not.toBeNull()
    })
})
