// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, within} from "@testing-library/react"
import {PlayerInfo, PlayerInfoBackend, PlayerLine} from "../../backends/player.ts"
import PlayerCard from "./PlayerCard.tsx"

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

const ana: PlayerLine = {name: "Ana", countryCode: "fr", guest: false, admin: false}
const bo: PlayerLine = {name: "guest_Bo", countryCode: "de", guest: true, admin: false}

const backendAnswering = (answer: () => Promise<PlayerInfo | undefined>) =>
    ({playerInfo: vi.fn(answer)}) satisfies PlayerInfoBackend

const stat = (label: string) => screen.getByText(label).nextElementSibling?.textContent

describe("PlayerCard", () => {
    it("shows who was clicked, and the player's stats once read", async () => {
        const backend = backendAnswering(async () => ({
            name: "Ana", tilesTaken: 1234, streakCurrent: 1, streakBest: 7, createdAt: Date.UTC(2026, 8, 1, 12), admin: false,
        }))
        render(<PlayerCard player={ana} backend={backend} onClose={() => {}}/>)

        const dialog = screen.getByRole("dialog", {name: "Ana"})
        expect(within(dialog).getByRole("img", {name: "France"})).toBeDefined()
        expect(dialog.textContent).not.toContain("#")
        expect(within(dialog).getByRole("status").textContent).toBe("Loading…")

        await screen.findByText("Tiles taken")
        expect(stat("Tiles taken")).toBe((1234).toLocaleString())
        expect(stat("Streak")).toBe("1 day")
        expect(stat("Best streak")).toBe("7 days")
        expect(stat("Playing since")).toBe(new Intl.DateTimeFormat(undefined, {dateStyle: "medium"}).format(Date.UTC(2026, 8, 1, 12)))
        expect(backend.playerInfo).toHaveBeenCalledWith("Ana")
    })

    it("leaves out a creation date the server does not know", async () => {
        render(<PlayerCard player={ana}
                           backend={backendAnswering(async () => ({name: "Ana", tilesTaken: 0, streakCurrent: 0, streakBest: 0, admin: false}))}
                           onClose={() => {}}/>)

        await screen.findByText("Tiles taken")
        expect(screen.queryByText("Playing since")).toBeNull()
    })

    it("asks nothing for a guest, and says it has no stats", () => {
        const backend = backendAnswering(async () => undefined)
        render(<PlayerCard player={bo} backend={backend} onClose={() => {}}/>)

        expect(screen.getByText(/Guests have no stats/)).toBeDefined()
        expect(backend.playerInfo).not.toHaveBeenCalled()
    })

    it("says so when no player holds the name any more", async () => {
        render(<PlayerCard player={ana} backend={backendAnswering(async () => undefined)} onClose={() => {}}/>)

        expect(await screen.findByText("No player holds this name now.")).toBeDefined()
    })

    it("says so when the stats cannot be read", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        render(<PlayerCard player={ana}
                           backend={backendAnswering(async () => Promise.reject(new Error("down")))}
                           onClose={() => {}}/>)

        expect(await screen.findByText("The stats could not be loaded.")).toBeDefined()
    })

    it("crowns an admin in its title, as clicked or as read", async () => {
        render(<PlayerCard player={{...ana, admin: true}}
                           backend={backendAnswering(async () => undefined)}
                           onClose={() => {}}/>)
        expect(within(screen.getByRole("dialog")).getByRole("img", {name: "Admin"})).toBeDefined()
        cleanup()

        render(<PlayerCard player={ana}
                           backend={backendAnswering(async () => ({name: "Ana", tilesTaken: 0, streakCurrent: 0, streakBest: 0, admin: true}))}
                           onClose={() => {}}/>)
        expect(screen.queryByRole("img", {name: "Admin"})).toBeNull()
        expect(await screen.findByRole("img", {name: "Admin"})).toBeDefined()
    })
})
