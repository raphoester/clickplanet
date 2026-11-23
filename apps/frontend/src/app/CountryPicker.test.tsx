// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import CountryPicker from "./CountryPicker.tsx"
import {useFlags} from "./flags/flagsStore.ts"
import type {ListedFlag} from "../domain/flags.ts"

const BAVARIA: ListedFlag = {code: "r-k3p9x2ma0b", name: "Bavaria", closed: false}
const SAVOY: ListedFlag = {code: "r-0a1b2c3d4e", name: "Savoy", closed: true}

afterEach(() => {
    cleanup()
    useFlags.getState().serve(undefined)
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
})
