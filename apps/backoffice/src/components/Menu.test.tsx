// @vitest-environment jsdom
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {useState} from "react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {PageId, titleOf} from "../navigation.ts"
import {Menu} from "./Menu.tsx"

afterEach(cleanup)

function Shown({onGo = () => undefined}: {onGo?: (page: PageId) => void}) {
    const [page, setPage] = useState<PageId>("notes")
    return (
        <>
            <Menu page={page} onGo={to => { setPage(to); onGo(to) }}/>
            <p>{titleOf(page)}</p>
        </>
    )
}

const burger = () => screen.getByRole("button", {name: "Menu"})

describe("Menu", () => {
    it("opens from the burger, with every page by group", () => {
        render(<Shown/>)
        expect(burger().getAttribute("aria-expanded")).toBe("false")
        expect(screen.queryByRole("dialog")).toBeNull()

        fireEvent.click(burger())

        const drawer = screen.getByRole("dialog", {name: "Menu"})
        expect(burger().getAttribute("aria-expanded")).toBe("true")
        expect(within(drawer).getAllByRole("heading").map(h => h.textContent))
            .toEqual(["Overview", "Feedback", "Moderation", "Map", "Ops"])
        expect(within(drawer).getByRole("button", {name: "Ideas and bugs"}).getAttribute("aria-current")).toBe("page")
        expect(document.activeElement).toBe(within(drawer).getByRole("button", {name: "Ideas and bugs"}))
    })

    it("goes to the page picked, and closes", () => {
        const onGo = vi.fn()
        render(<Shown onGo={onGo}/>)

        fireEvent.click(burger())
        fireEvent.click(screen.getByRole("button", {name: "Paint random tiles"}))

        expect(onGo).toHaveBeenCalledWith("paint")
        expect(screen.getByText("Paint random tiles")).toBeTruthy()
        expect(screen.queryByRole("dialog")).toBeNull()
        expect(document.activeElement).toBe(burger())
    })

    it("closes on Escape, on its close button and on a click beside it", () => {
        render(<Shown/>)

        fireEvent.click(burger())
        fireEvent.keyDown(document, {key: "Escape"})
        expect(screen.queryByRole("dialog")).toBeNull()

        fireEvent.click(burger())
        fireEvent.click(screen.getByRole("button", {name: "Close the menu"}))
        expect(screen.queryByRole("dialog")).toBeNull()

        fireEvent.click(burger())
        fireEvent.pointerDown(document.body)
        expect(screen.queryByRole("dialog")).toBeNull()
    })

    it("keeps the focus in the drawer while it is open", () => {
        render(<Shown/>)
        fireEvent.click(burger())
        const drawer = screen.getByRole("dialog")
        const buttons = within(drawer).getAllByRole("button")

        buttons[buttons.length - 1].focus()
        fireEvent.keyDown(document, {key: "Tab"})
        expect(document.activeElement).toBe(buttons[0])

        fireEvent.keyDown(document, {key: "Tab", shiftKey: true})
        expect(document.activeElement).toBe(buttons[buttons.length - 1])
    })
})
