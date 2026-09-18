// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {ChatMessage} from "../../backends/chat.ts"
import ChatLog from "./ChatLog.tsx"

afterEach(cleanup)

const message = (id: string, authorName: string, countryCode: string): ChatMessage =>
    ({id, sentAt: Date.UTC(2026, 8, 17, 12), authorName, authorAdmin: false, countryCode, text: "hello", reactions: [], reactionsVersion: 0})

describe("ChatLog", () => {
    // jsdom lays nothing out, so the log's scroll position is never read.
    it("opens the author of a message from its name, and tells a guest by its prefix", async () => {
        const onOpenPlayer = vi.fn()
        render(<ChatLog loading={false}
                        onOpenPlayer={onOpenPlayer}
                        messages={[message("1", "Ana", "fr"), message("2", "guest_Bo", "de")]}/>)

        await userEvent.click(screen.getByRole("button", {name: "Ana"}))
        await userEvent.click(screen.getByRole("button", {name: "guest_Bo"}))

        expect(onOpenPlayer.mock.calls).toEqual([
            [{name: "Ana", countryCode: "fr", guest: false, admin: false}],
            [{name: "guest_Bo", countryCode: "de", guest: true, admin: false}],
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
        expect(onOpenPlayer).toHaveBeenCalledWith({name: "Ana", countryCode: "fr", guest: false, admin: true})
    })
})
