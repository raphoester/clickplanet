// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import SelectWithSearch from "./SelectWithSearch.tsx"

const VALUES = [
    {code: "fr", name: "France"},
    {code: "jp", name: "Japan"},
    {code: "de", name: "Germany"},
]

const optionNames = () => screen.queryAllByRole("option").map(o => o.textContent?.trim())

afterEach(cleanup)

describe("SelectWithSearch", () => {
    it("lists every value", () => {
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={vi.fn()}/>)
        expect(optionNames()).toEqual(["France", "Japan", "Germany"])
    })

    it("filters the list as you type, case-insensitively", async () => {
        const user = userEvent.setup()
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={vi.fn()}/>)

        await user.type(screen.getByPlaceholderText("🔍 Search..."), "jAp")
        expect(optionNames()).toEqual(["Japan"])
    })

    it("reports the chosen value to its caller", async () => {
        const user = userEvent.setup()
        const onChange = vi.fn()
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={onChange}/>)

        await user.selectOptions(screen.getByRole("listbox"), "jp")
        expect(onChange).toHaveBeenCalledWith(VALUES[1])
    })

    /**
     * The selection is the caller's state. This component used to copy it into
     * its own on mount and never resync, so it showed a stale country whenever
     * the selection changed from anywhere else.
     */
    it("follows the selection it is given", () => {
        const {rerender} = render(
            <SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={vi.fn()}/>)
        expect(screen.getByRole("listbox")).toHaveProperty("value", "fr")

        rerender(<SelectWithSearch values={VALUES} selected={VALUES[1]} onChange={vi.fn()}/>)
        expect(screen.getByRole("listbox")).toHaveProperty("value", "jp")
    })

    it("clears the search once something is chosen", async () => {
        const user = userEvent.setup()
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={vi.fn()}/>)

        const search = screen.getByPlaceholderText("🔍 Search...")
        await user.type(search, "jap")
        await user.selectOptions(screen.getByRole("listbox"), "jp")

        expect(search).toHaveProperty("value", "")
        expect(optionNames()).toEqual(["France", "Japan", "Germany"])
    })
})
