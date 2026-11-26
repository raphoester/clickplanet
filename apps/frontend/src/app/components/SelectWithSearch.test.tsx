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

    it("stands a flag in front of every option", () => {
        const {container} = render(
            <SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={vi.fn()}/>)

        expect(container.querySelectorAll(".input-select-option .country-flag"))
            .toHaveLength(VALUES.length)
    })

    it("filters the list as you type, case-insensitively", async () => {
        const user = userEvent.setup()
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

    it("moves through the list with the arrow keys and commits on Enter", async () => {
        const user = userEvent.setup()
        const onChange = vi.fn()
        render(<SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={onChange}/>)

        const list = screen.getByRole("listbox")
        list.focus()
        await user.keyboard("{ArrowDown}{ArrowDown}{Enter}")

        expect(onChange).toHaveBeenCalledWith(VALUES[2])
    })

    it("follows the selection it is given", () => {
        const {rerender} = render(
            <SelectWithSearch values={VALUES} selected={VALUES[0]} onChange={vi.fn()}/>)
        expect(selectedName()).toEqual(["France"])

        rerender(<SelectWithSearch values={VALUES} selected={VALUES[1]} onChange={vi.fn()}/>)
        expect(selectedName()).toEqual(["Japan"])
    })

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

    it("shows its footer under the list, and in its place what a search found no value for", async () => {
        const user = userEvent.setup()
        render(<SelectWithSearch values={VALUES}
                                 selected={VALUES[0]}
                                 onChange={vi.fn()}
                                 footer={<p>footer</p>}
                                 noMatch={(search) => <p>{`none for ${search}`}</p>}/>)
        const search = screen.getByPlaceholderText("Search a country")
        expect(screen.getByText("footer")).not.toBeNull()

        await user.type(search, "Fra")
        expect(screen.getByText("footer"), "the selection matches").not.toBeNull()

        await user.clear(search)
        await user.type(search, "Bavaria")
        expect(screen.queryByText("footer")).toBeNull()
        expect(screen.getByText("none for Bavaria")).not.toBeNull()
        expect(optionNames(), "the selection stays").toEqual(["France"])
    })
})
