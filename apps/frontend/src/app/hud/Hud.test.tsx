// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, fireEvent, render, screen} from "@testing-library/react"
import Sheet from "./Sheet.tsx"
import TabBar from "./TabBar.tsx"
import StatusBar, {STATUS_BOTTOM} from "./StatusBar.tsx"
import MenuPanel from "../components/MenuPanel.tsx"
import {Countries} from "../../domain/countries.ts"

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

describe("TabBar", () => {
    it("offers the board, the chat, the account and More, in that order", () => {
        render(<TabBar onOpen={vi.fn()} chat unread={0} you="player"/>)
        expect(screen.getAllByRole("button").map((b) => b.textContent)).toEqual(["Board", "Chat", "You", "More"])
    })

    it("names the account Sign in for a guest", () => {
        render(<TabBar onOpen={vi.fn()} chat unread={0} you="guest"/>)
        expect(screen.getByRole("button", {name: "Sign in"})).toBeDefined()
    })

    it("leaves out what is not wired", () => {
        render(<TabBar onOpen={vi.fn()} chat={false} unread={0}/>)
        expect(screen.getAllByRole("button").map((b) => b.textContent)).toEqual(["Board", "More"])
    })

    it("counts what the chat missed", () => {
        render(<TabBar onOpen={vi.fn()} chat unread={3}/>)
        expect(screen.getByRole("button", {name: "Chat, 3 new messages"}).textContent).toBe("Chat3")
    })

    it("says which sheet is open, and opens the one pressed", () => {
        const onOpen = vi.fn()
        render(<TabBar open="board" onOpen={onOpen} chat unread={0}/>)

        expect(screen.getByRole("button", {name: "Board"}).getAttribute("aria-expanded")).toBe("true")
        expect(screen.getByRole("button", {name: "More"}).getAttribute("aria-expanded")).toBe("false")

        fireEvent.click(screen.getByRole("button", {name: "More"}))
        expect(onOpen).toHaveBeenCalledWith("more")
    })
})

describe("Sheet", () => {
    it("is a region named by its title, with focus on it", () => {
        render(<Sheet title="Leaderboard" onClose={vi.fn()}><p>rows</p></Sheet>)

        const heading = screen.getByRole("heading", {name: "Leaderboard"})
        expect(screen.getByRole("region", {name: "Leaderboard"})).toBeDefined()
        expect(document.activeElement).toBe(heading)
    })

    it("closes on its × and on Escape", () => {
        const onClose = vi.fn()
        render(<Sheet title="More" onClose={onClose}><p>tiles</p></Sheet>)

        fireEvent.click(screen.getByRole("button", {name: "Close"}))
        fireEvent.keyDown(document, {key: "Escape"})
        expect(onClose).toHaveBeenCalledTimes(2)
    })

    it("steps back out of a panel inside it on Escape, rather than closing", () => {
        const onClose = vi.fn()
        const onBack = vi.fn()
        const {rerender} = render(<Sheet title="More" onClose={onClose}><p>tiles</p></Sheet>)
        rerender(<Sheet title="More" onClose={onClose}>
            <MenuPanel title="Sound" onClose={onBack}><p>switches</p></MenuPanel>
        </Sheet>)

        fireEvent.keyDown(document, {key: "Escape"})
        expect(onBack).toHaveBeenCalledTimes(1)
        expect(onClose).not.toHaveBeenCalled()
    })
})

describe("StatusBar", () => {
    const france = Countries.get("fr")!

    it("names the country and its rank, and opens the board", () => {
        const onOpenBoard = vi.fn()
        render(<StatusBar country={france} rank={3} boardOpen={false} onOpenBoard={onOpenBoard}/>)

        fireEvent.click(screen.getByRole("button", {name: "France, rank 3. Leaderboard"}))
        expect(onOpenBoard).toHaveBeenCalledTimes(1)
    })

    it("says when the country has no rank yet", () => {
        render(<StatusBar country={france} rank={null} boardOpen={false} onOpenBoard={vi.fn()}/>)
        expect(screen.getByRole("button", {name: "France, no rank yet. Leaderboard"})).toBeDefined()
    })

    it("tells the moments at the top where it ends", () => {
        vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue(new DOMRect(0, 0, 390, 62))

        const {unmount} = render(<StatusBar country={france} rank={1} boardOpen={false} onOpenBoard={vi.fn()}/>)
        expect(document.documentElement.style.getPropertyValue(STATUS_BOTTOM)).toBe("62px")

        unmount()
        expect(document.documentElement.style.getPropertyValue(STATUS_BOTTOM)).toBe("")
    })
})
