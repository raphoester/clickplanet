// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {ChatAnnouncement, ChatMessage} from "../../backends/chat.ts"
import ChatLog from "./ChatLog.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import {NameColor} from "../../backends/player.ts"

afterEach(cleanup)

const flagOf = (code: string) => render(<CountryFlag code={code}/>).container.innerHTML

const message = (id: string, authorName: string, countryCode: string): ChatMessage =>
    ({id, sentAt: Date.UTC(2026, 8, 17, 12), authorName, authorAdmin: false, authorColor: NameColor.UNSPECIFIED, authorStreak: 0, countryCode, text: "hello", reactions: [], reactionsVersion: 0})

describe("ChatLog", () => {
    it("opens the author of a message from its name, and tells a guest by its prefix", async () => {
        const onOpenPlayer = vi.fn()
        render(<ChatLog loading={false}
                        onOpenPlayer={onOpenPlayer}
                        messages={[message("1", "Ana", "fr"), message("2", "guest_Bo", "de")]}/>)

        await userEvent.click(screen.getByRole("button", {name: "Ana"}))
        await userEvent.click(screen.getByRole("button", {name: "guest_Bo"}))

        expect(onOpenPlayer.mock.calls).toEqual([
            [{name: "Ana", countryCode: "fr", guest: false, admin: false, color: NameColor.UNSPECIFIED, streak: 0}],
            [{name: "guest_Bo", countryCode: "de", guest: true, admin: false, color: NameColor.UNSPECIFIED, streak: 0}],
        ])
    })

    it("shows a guest's name once, and nothing of its address", () => {
        const {container} = render(<ChatLog loading={false} messages={[message("1", "guest_a1b2c3", "de")]}/>)

        expect(container.querySelector(".chat-message-head")?.textContent).not.toContain("#")
        expect(screen.getAllByText(/a1b2c3/)).toHaveLength(1)
    })

    it("shows a plain name with nothing to open", () => {
        render(<ChatLog loading={false} messages={[message("1", "Ana", "fr")]}/>)

        expect(screen.queryByRole("button", {name: "Ana"})).toBeNull()
        expect(screen.getByText("Ana")).toBeDefined()
    })

    it("crowns an admin's message, and keeps it on the player it opens", async () => {
        const onOpenPlayer = vi.fn()
        const admin = {...message("1", "Ana", "fr"), authorAdmin: true}
        render(<ChatLog loading={false} onOpenPlayer={onOpenPlayer} messages={[admin, message("2", "kiran_07", "in")]}/>)

        expect(screen.getAllByRole("img", {name: "Admin"})).toHaveLength(1)
        await userEvent.click(screen.getByRole("button", {name: "Ana"}))
        expect(onOpenPlayer).toHaveBeenCalledWith({
            name: "Ana", countryCode: "fr", guest: false, admin: true, color: NameColor.UNSPECIFIED, streak: 0,
        })
    })

    it("lights a flame beside a streak of three days or more, and keeps it on the player it opens", async () => {
        const onOpenPlayer = vi.fn()
        const burning = {...message("1", "Ana", "fr"), authorStreak: 12}
        const starting = {...message("2", "kiran_07", "in"), authorStreak: 2}
        render(<ChatLog loading={false} onOpenPlayer={onOpenPlayer} messages={[burning, starting]}/>)

        expect(screen.getAllByRole("img", {name: /streak/})).toHaveLength(1)
        expect(screen.getByRole("img", {name: "12-day streak"}).textContent).toBe("12")
        await userEvent.click(screen.getByRole("button", {name: "Ana"}))
        expect(onOpenPlayer).toHaveBeenCalledWith(expect.objectContaining({streak: 12}))
    })

    it("shows the title its author wears beside the name, and keeps it on the player it opens", async () => {
        const onOpenPlayer = vi.fn()
        const warlord = {id: "warlord", name: "Warlord", rank: {trackId: "conquest", trackName: "Conquest", number: 3, count: 5}}
        const dressed = {...message("1", "Ana", "fr"), authorTitle: warlord}
        const bare = message("2", "kiran_07", "in")
        render(<ChatLog loading={false} onOpenPlayer={onOpenPlayer} messages={[dressed, bare]}/>)

        expect(screen.getAllByRole("img", {name: "Warlord"})).toHaveLength(1)
        expect(screen.getByRole("img", {name: "Warlord"}).querySelector(".title-emblem")).not.toBeNull()
        await userEvent.click(screen.getByRole("button", {name: "Ana"}))
        expect(onOpenPlayer).toHaveBeenCalledWith(expect.objectContaining({wornTitle: warlord}))
    })

    it("paints a name in the color its player chose, and grey with none or as a guest", () => {
        const chosen = {...message("1", "Ana", "fr"), authorColor: NameColor.TEAL}
        const guest = {...message("2", "guest_Bo", "de"), authorColor: NameColor.TEAL}
        const none = message("3", "Cy", "it")
        const {container} = render(<ChatLog loading={false} messages={[chosen, guest, none]}/>)

        const [ana, bo, cy] = [...container.querySelectorAll<HTMLElement>(".chat-message")]
        expect(ana.style.getPropertyValue("--author-hue")).toBe("165")
        expect(ana.style.getPropertyValue("--author-chroma")).toBe("")
        expect(bo.style.getPropertyValue("--author-chroma")).toBe("0")
        expect(cy.style.getPropertyValue("--author-chroma")).toBe("0")
    })

    it("says which country passed the leader, under the new leader's flag", () => {
        const lead: ChatAnnouncement = {
            kind: "leadChanged", id: "lead", announcedAt: Date.UTC(2026, 8, 17, 12), season: 0, leader: "bg", passed: "fr",
        }
        const {container} = render(<ChatLog loading={false} messages={[]} announcements={[lead]}/>)

        const line = container.querySelector("li")!
        expect(line.className).toBe("chat-announcement")
        expect(line.querySelector(".chat-announcement-text")?.textContent).toBe("Bulgaria passes France")
        expect(line.querySelector(".country-flag")?.outerHTML).toBe(flagOf("bg"))
    })

    it("names the winner of a season, under the winner's flag", () => {
        const won: ChatAnnouncement = {
            kind: "seasonWon", id: "won", announcedAt: Date.UTC(2026, 8, 17, 12), season: 0, winner: "dz",
        }
        const {container} = render(<ChatLog loading={false} messages={[]} announcements={[won]}/>)

        const line = container.querySelector("li")!
        expect(line.className).toBe("chat-announcement chat-announcement--won")
        expect(line.querySelector(".chat-announcement-text")?.textContent).toBe("Algeria wins Season 0")
        expect(line.querySelector(".country-flag")?.outerHTML).toBe(flagOf("dz"))
    })
})
