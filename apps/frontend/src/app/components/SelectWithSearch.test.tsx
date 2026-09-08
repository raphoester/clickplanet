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
const selectedName = () =>
    screen.queryAllByRole("option", {selected: true}).map(o => o.textContent?.trim())

afterEach(cleanup)

describe("SelectWithSearch", () => {
    it("lists every value", () => {
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={vi.fn()}/>)
        expect(optionNames()).toEqual(["France", "Japan", "Germany"])
    })

    it("filters the list as you type, case-insensitively", async () => {
        const user = userEvent.setup()
        // Selecting Japan keeps the assertion about filtering alone; the
        // selection is kept in the list regardless, which its own tests cover.
        render(<SelectWithSearch values={VALUES} selected={VALUES[1]} onChange={vi.fn()}/>)

        await user.type(screen.getByPlaceholderText("Search a country"), "jAp")
        expect(optionNames()).toEqual(["Japan"])
    })

    it("reports the chosen value to its caller", async () => {
        const user = userEvent.setup()
        const onChange = vi.fn()
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={onChange}/>)

        await user.click(screen.getByRole("option", {name: "Japan"}))
        expect(onChange).toHaveBeenCalledWith(VALUES[1])
    })

    /**
     * Every option is visible at once, tapped directly: mobile WebKit collapses
     * a `<select size={n}>` into a native picker showing one row, which is why
     * this list is our own markup.
     */
    it("shows the whole list rather than a native select", () => {
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={vi.fn()}/>)

        expect(screen.queryByRole("combobox")).toBeNull()
        expect(screen.getAllByRole("option")).toHaveLength(VALUES.length)
    })

    it("moves through the list with the arrow keys and commits on Enter", async () => {
        const user = userEvent.setup()
        const onChange = vi.fn()
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={onChange}/>)

        const list = screen.getByRole("listbox")
        list.focus()
        await user.keyboard("{ArrowDown}{ArrowDown}{Enter}")

        expect(onChange).toHaveBeenCalledWith(VALUES[2])
    })

    it("picks the top match when Enter is pressed in the search field", async () => {
        const user = userEvent.setup()
        const onChange = vi.fn()
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={onChange}/>)

        await user.type(screen.getByPlaceholderText("Search a country"), "germ{Enter}")
        expect(onChange).toHaveBeenCalledWith(VALUES[2])
    })

    /**
     * The selection is the caller's state. This component used to copy it into
     * its own on mount and never resync, so it showed a stale country whenever
     * the selection changed from anywhere else.
     */
    it("follows the selection it is given", () => {
        const {rerender} = render(
            <SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={vi.fn()}/>)
        expect(selectedName()).toEqual(["France"])

        rerender(<SelectWithSearch values={VALUES} selected={VALUES[1]} onChange={vi.fn()}/>)
        expect(selectedName()).toEqual(["Japan"])
    })

    /**
     * Kept from #14, which fixed this for the `<select>` this list used to be: a
     * select whose value matched none of its options had the browser pick the
     * first one instead, so clicking the top result of a search fired no change
     * event and picked nothing. The markup that caused it is gone; the guarantee
     * it forced — the selection stays in the list, and every result is pickable
     * — is the behaviour worth keeping.
     */
    it("keeps the selection in the list while the search filters it out", async () => {
        const user = userEvent.setup()
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={vi.fn()}/>)

        await user.type(screen.getByPlaceholderText("Search a country"), "jap")

        expect(selectedName()).toEqual(["France"])
        expect(optionNames()).toEqual(["France", "Japan"])
    })

    it("reports the only match of a search, which is otherwise the first option", async () => {
        const user = userEvent.setup()
        const onChange = vi.fn()
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={onChange}/>)

        await user.type(screen.getByPlaceholderText("Search a country"), "jap")
        await user.click(screen.getByRole("option", {name: "Japan"}))

        expect(onChange).toHaveBeenCalledWith(VALUES[1])
    })

    it("reports the first search result when several match", async () => {
        const user = userEvent.setup()
        const onChange = vi.fn()
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={onChange}/>)

        await user.type(screen.getByPlaceholderText("Search a country"), "an")
        expect(optionNames()).toEqual(["France", "Japan", "Germany"])

        await user.click(screen.getByRole("option", {name: "Japan"}))
        expect(onChange).toHaveBeenCalledWith(VALUES[1])
    })

    /** The keyboard form of the same bug: Enter must not re-pick the pinned row. */
    it("puts the keyboard cursor on the first match, not on the pinned selection", async () => {
        const user = userEvent.setup()
        const onChange = vi.fn()
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={onChange}/>)

        await user.type(screen.getByPlaceholderText("Search a country"), "jap{Enter}")

        expect(onChange).toHaveBeenCalledWith(VALUES[1])
    })

    it("clears the search once something is chosen", async () => {
        const user = userEvent.setup()
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={vi.fn()}/>)

        const search = screen.getByPlaceholderText("Search a country")
        await user.type(search, "jap")
        await user.click(screen.getByRole("option", {name: "Japan"}))

        expect(search).toHaveProperty("value", "")
        expect(optionNames()).toEqual(["France", "Japan", "Germany"])
    })
})
