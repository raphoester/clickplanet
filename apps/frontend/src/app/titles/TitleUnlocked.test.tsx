// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {PlayerTitle} from "../../backends/player.ts"
import TitleUnlocked from "./TitleUnlocked.tsx"

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

const raider: PlayerTitle = {id: "raider", name: "Raider", rank: {trackId: "conquest", trackName: "Conquest", number: 2, count: 5}}

describe("TitleUnlocked", () => {
    it("names the title and its rank", () => {
        render(<TitleUnlocked title={raider} onClose={vi.fn()}/>)

        const dialog = screen.getByRole("dialog", {name: "New title unlocked"})
        expect(within(dialog).getByText("Raider")).toBeDefined()
        expect(within(dialog).getByText("Rank 2 of 5 · Conquest")).toBeDefined()
    })

    it("says no rank for a title on no track", () => {
        render(<TitleUnlocked title={{id: "og", name: "OG"}} onClose={vi.fn()}/>)

        expect(screen.queryByText(/Rank/)).toBeNull()
    })

    it("wears the title, then closes", async () => {
        const onWear = vi.fn(async () => raider)
        const onClose = vi.fn()
        render(<TitleUnlocked title={raider} onWear={onWear} onClose={onClose}/>)

        await userEvent.click(screen.getByRole("button", {name: "Wear it"}))

        expect(onWear).toHaveBeenCalledWith("raider")
        await vi.waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    })

    it("stays open when the title could not be worn", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const onClose = vi.fn()
        render(<TitleUnlocked title={raider} onWear={vi.fn(async () => Promise.reject(new Error("down")))} onClose={onClose}/>)

        await userEvent.click(screen.getByRole("button", {name: "Wear it"}))

        await vi.waitFor(() => expect(screen.getByRole("button", {name: "Wear it"})).toHaveProperty("disabled", false))
        expect(onClose).not.toHaveBeenCalled()
    })

    it("offers only to close when nothing can wear it", async () => {
        const onClose = vi.fn()
        render(<TitleUnlocked title={raider} onClose={onClose}/>)

        expect(screen.queryByRole("button", {name: "Wear it"})).toBeNull()
        await userEvent.click(screen.getByText("Close"))
        expect(onClose).toHaveBeenCalled()
    })
})
