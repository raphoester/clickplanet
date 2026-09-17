// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {ChatMessage} from "../../backends/chat.ts"
import ChatLog from "./ChatLog.tsx"

afterEach(cleanup)

const message = (id: string, authorName: string, authorTag: string, countryCode: string): ChatMessage =>
    ({id, sentAt: Date.UTC(2026, 8, 17, 12), authorName, authorTag, countryCode, text: "hello"})

describe("ChatLog", () => {
    // jsdom lays nothing out, so the log's scroll position is never read.
    it("opens the author of a message from its name, and tells a guest by its prefix", async () => {
        const onOpenPlayer = vi.fn()
        render(<ChatLog loading={false}
                        onOpenPlayer={onOpenPlayer}
                        messages={[message("1", "Ana", "4f2ca1", "fr"), message("2", "guest_Bo", "91aa3d", "de")]}/>)

        await userEvent.click(screen.getByRole("button", {name: "Ana"}))
        await userEvent.click(screen.getByRole("button", {name: "guest_Bo"}))

        expect(onOpenPlayer.mock.calls).toEqual([
            [{name: "Ana", tag: "4f2ca1", countryCode: "fr", guest: false}],
            [{name: "guest_Bo", tag: "91aa3d", countryCode: "de", guest: true}],
        ])
    })

    it("shows a plain name with nothing to open", () => {
        render(<ChatLog loading={false} messages={[message("1", "Ana", "4f2ca1", "fr")]}/>)

        expect(screen.queryByRole("button", {name: "Ana"})).toBeNull()
        expect(screen.getByText("Ana")).toBeDefined()
    })
})
