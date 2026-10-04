// @vitest-environment jsdom
import {useState} from "react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
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

const at = (top: number, bottom: number, left: number) => vi.spyOn(button(), "getBoundingClientRect")
    .mockReturnValue({top, bottom, left, right: left + 120, width: 120, height: bottom - top, x: left, y: top, toJSON: () => ({})})

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

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

    it("opens under its button, over whatever is around it", async () => {
        const {user} = shown()
        at(100, 130, 40)

        await user.click(button())

        expect(list()!.style.top).toBe("130px")
        expect(list()!.style.left).toBe("40px")
    })

    it("opens over its button when there is no room under it", async () => {
        const {user} = shown()
        at(window.innerHeight - 50, window.innerHeight - 20, 40)
        vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockReturnValue(144)

        await user.click(button())

        expect(list()!.style.top).toBe(`${window.innerHeight - 50 - 144}px`)
    })

    it("closes when what holds it scrolls, and only then", async () => {
        const {user} = shown()

        await user.click(button())
        act(() => {
            screen.getByText("elsewhere").dispatchEvent(new Event("scroll"))
        })
        expect(list()).not.toBeNull()

        act(() => {
            document.dispatchEvent(new Event("scroll"))
        })
        expect(list()).toBeNull()
    })
})
