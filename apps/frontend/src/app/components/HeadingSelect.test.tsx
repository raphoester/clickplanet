// @vitest-environment jsdom
import {useState} from "react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import HeadingSelect, {HeadingChoice} from "./HeadingSelect.tsx"

type Fruit = "apple" | "pear" | "plum"

const CHOICES: HeadingChoice<Fruit>[] = [
    {value: "apple", name: "Apple", label: "Apple"},
    {value: "pear", name: "Pear", label: "Pear"},
    {value: "plum", name: "Plum", label: <span>Plum</span>},
]

function Harness({onChange}: { onChange: (value: Fruit) => void }) {
    const [value, setValue] = useState<Fruit>("apple")
    return <>
        <HeadingSelect label="Fruit" choices={CHOICES} value={value} onChange={(chosen) => {
            onChange(chosen)
            setValue(chosen)
        }}/>
        <p>elsewhere</p>
    </>
}

function shown() {
    const onChange = vi.fn()
    render(<Harness onChange={onChange}/>)
    return {onChange, user: userEvent.setup()}
}

const button = () => screen.getByRole("button", {name: /^Fruit: /})
const list = () => screen.queryByRole("listbox", {name: "Fruit"})
const active = () => document.getElementById(list()!.getAttribute("aria-activedescendant")!)!.textContent

afterEach(cleanup)

describe("HeadingSelect", () => {
    it("names the choice it shows, closed", () => {
        shown()

        expect(button().getAttribute("aria-label")).toBe("Fruit: Apple")
        expect(button().getAttribute("aria-expanded")).toBe("false")
        expect(list()).toBeNull()
    })

    it("opens on a press, on the choice it shows", async () => {
        const {user} = shown()

        await user.click(button())

        expect(button().getAttribute("aria-expanded")).toBe("true")
        expect(screen.getAllByRole("option").map((option) => option.textContent)).toEqual(["Apple", "Pear", "Plum"])
        expect(screen.getByRole("option", {selected: true}).textContent).toBe("Apple")
        expect(document.activeElement).toBe(list())
        expect(active()).toBe("Apple")
    })

    it("chooses on a press, closes, and gives the focus back", async () => {
        const {user, onChange} = shown()

        await user.click(button())
        await user.click(screen.getByRole("option", {name: "Plum"}))

        expect(onChange).toHaveBeenCalledWith("plum")
        expect(list()).toBeNull()
        expect(button().getAttribute("aria-label")).toBe("Fruit: Plum")
        expect(document.activeElement).toBe(button())
    })

    it("is driven by the keys", async () => {
        const {user, onChange} = shown()

        button().focus()
        await user.keyboard("{ArrowDown}")
        expect(active()).toBe("Apple")

        await user.keyboard("{ArrowDown}{ArrowDown}{ArrowDown}")
        expect(active()).toBe("Plum")

        await user.keyboard("{Home}")
        expect(active()).toBe("Apple")

        await user.keyboard("{End}{ArrowUp}{Enter}")
        expect(onChange).toHaveBeenCalledWith("pear")
        expect(list()).toBeNull()
        expect(document.activeElement).toBe(button())
    })

    it("closes on Escape and changes nothing", async () => {
        const {user, onChange} = shown()

        await user.click(button())
        await user.keyboard("{ArrowDown}{Escape}")

        expect(list()).toBeNull()
        expect(onChange).not.toHaveBeenCalled()
        expect(document.activeElement).toBe(button())
    })

    it("closes on a press elsewhere", async () => {
        const {user, onChange} = shown()

        await user.click(button())
        await user.click(screen.getByText("elsewhere"))

        expect(list()).toBeNull()
        expect(onChange).not.toHaveBeenCalled()
    })

    it("closes on a press of its button", async () => {
        const {user} = shown()

        await user.click(button())
        await user.click(button())

        expect(list()).toBeNull()
    })
})
