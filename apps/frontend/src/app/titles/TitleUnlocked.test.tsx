// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {PlayerTitle} from "../../backends/player.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import TitleUnlocked from "./TitleUnlocked.tsx"

beforeEach(() => {
    vi.useFakeTimers({shouldAdvanceTime: true})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
})

const raider: PlayerTitle = {id: "raider", name: "Raider", rank: {trackId: "conquest", trackName: "Conquest", number: 2, count: 5}}

const user = () => userEvent.setup({advanceTimers: vi.advanceTimersByTime})

function reveal() {
    act(() => vi.advanceTimersByTime(TITLE_REVEAL.ready * 1000))
}

describe("TitleUnlocked", () => {
    it("names the title and its rank", () => {
        render(<TitleUnlocked title={raider} play={vi.fn()} onClose={vi.fn()}/>)

        const dialog = screen.getByRole("dialog", {name: "New title! Raider"})
        expect(within(dialog).getByText("Rank 2 of 5 · Conquest")).toBeDefined()
    })

    it("says no rank for a title on no track", () => {
        render(<TitleUnlocked title={{id: "og", name: "OG"}} play={vi.fn()} onClose={vi.fn()}/>)

        expect(screen.queryByText(/Rank/)).toBeNull()
    })

    it("plays the fanfare as it opens", () => {
        const play = vi.fn()
        render(<TitleUnlocked title={raider} play={play} onClose={vi.fn()}/>)

        expect(play).toHaveBeenCalledWith("title")
    })

    it("holds its buttons back until the reveal has played", () => {
        render(<TitleUnlocked title={raider} play={vi.fn()} onWear={vi.fn(async () => raider)} onClose={vi.fn()}/>)

        expect(screen.queryByRole("button")).toBeNull()
        reveal()
        expect(screen.getByRole("button", {name: "Close"})).toBeDefined()
        expect(screen.getByRole("button", {name: "Wear it"})).toBeDefined()
    })

    it("stays open when a click lands around it", async () => {
        const onClose = vi.fn()
        render(<TitleUnlocked title={raider} play={vi.fn()} onClose={onClose}/>)
        const dialog = screen.getByRole("dialog")

        await user().click(dialog)
        reveal()
        await user().click(dialog)

        expect(onClose).not.toHaveBeenCalled()
    })

    it("closes on Escape only once the reveal has played", async () => {
        const onClose = vi.fn()
        render(<TitleUnlocked title={raider} play={vi.fn()} onClose={onClose}/>)

        await user().keyboard("{Escape}")
        expect(onClose).not.toHaveBeenCalled()

        reveal()
        await user().keyboard("{Escape}")
        expect(onClose).toHaveBeenCalledTimes(1)
    })

    it("wears the title, then closes", async () => {
        const onWear = vi.fn(async () => raider)
        const onClose = vi.fn()
        render(<TitleUnlocked title={raider} play={vi.fn()} onWear={onWear} onClose={onClose}/>)
        reveal()

        await user().click(screen.getByRole("button", {name: "Wear it"}))

        expect(onWear).toHaveBeenCalledWith("raider")
        await vi.waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    })

    it("stays open when the title could not be worn", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const onClose = vi.fn()
        render(<TitleUnlocked title={raider} play={vi.fn()} onWear={vi.fn(async () => Promise.reject(new Error("down")))} onClose={onClose}/>)
        reveal()

        await user().click(screen.getByRole("button", {name: "Wear it"}))

        await vi.waitFor(() => expect(screen.getByRole("button", {name: "Wear it"})).toHaveProperty("disabled", false))
        expect(onClose).not.toHaveBeenCalled()
    })

    it("offers only to close when nothing can wear it", async () => {
        const onClose = vi.fn()
        render(<TitleUnlocked title={raider} play={vi.fn()} onClose={onClose}/>)
        reveal()

        expect(screen.queryByRole("button", {name: "Wear it"})).toBeNull()
        await user().click(screen.getByRole("button", {name: "Close"}))
        expect(onClose).toHaveBeenCalled()
    })
})
