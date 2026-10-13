// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {ChatMessage} from "../../backends/chat.ts"
import {quoteOf} from "../../domain/chatLog.ts"
import ChatLog from "./ChatLog.tsx"
import {QUOTE_MAX_LENGTH} from "./QuoteLine.tsx"
import {NameColor} from "../../backends/player.ts"

afterEach(cleanup)

const message = (id: string, authorName: string, countryCode: string): ChatMessage =>
    ({id, sentAt: Date.UTC(2026, 8, 17, 12), authorAccountId: `account-${authorName}`, authorName, authorAdmin: false, authorColor: NameColor.UNSPECIFIED, authorStreak: 0, countryCode, text: "hello", reactions: [], reactionsVersion: 0})

describe("ChatLog", () => {
    it("opens the author of a message from its name with the message, and tells a guest by its prefix", async () => {
        const onOpenPlayer = vi.fn()
        render(<ChatLog loading={false}
                        onOpenPlayer={onOpenPlayer}
                        messages={[message("1", "Ana", "fr"), message("2", "guest_Bo", "de")]}/>)

        await userEvent.click(screen.getByRole("button", {name: "Ana"}))
        await userEvent.click(screen.getByRole("button", {name: "guest_Bo"}))

        expect(onOpenPlayer.mock.calls).toEqual([
            [{accountId: "account-Ana", name: "Ana", countryCode: "fr", guest: false, admin: false, color: NameColor.UNSPECIFIED, streak: 0}, "1"],
            [{accountId: "account-guest_Bo", name: "guest_Bo", countryCode: "de", guest: true, admin: false, color: NameColor.UNSPECIFIED, streak: 0}, "2"],
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
            accountId: "account-Ana", name: "Ana", countryCode: "fr", guest: false, admin: true, color: NameColor.UNSPECIFIED,
            streak: 0,
        }, "1")
    })

    it("lights a flame beside a streak of three days or more, and keeps it on the player it opens", async () => {
        const onOpenPlayer = vi.fn()
        const burning = {...message("1", "Ana", "fr"), authorStreak: 12}
        const starting = {...message("2", "kiran_07", "in"), authorStreak: 2}
        render(<ChatLog loading={false} onOpenPlayer={onOpenPlayer} messages={[burning, starting]}/>)

        expect(screen.getAllByRole("img", {name: /streak/})).toHaveLength(1)
        expect(screen.getByRole("img", {name: "12-day streak"}).textContent).toBe("12")
        await userEvent.click(screen.getByRole("button", {name: "Ana"}))
        expect(onOpenPlayer).toHaveBeenCalledWith(expect.objectContaining({streak: 12}), "1")
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
        expect(onOpenPlayer).toHaveBeenCalledWith(expect.objectContaining({wornTitle: warlord}), "1")
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

    describe("a reply", () => {
        const question = {...message("1", "Ana", "fr"), text: "who holds the north", authorColor: NameColor.TEAL}
        const answer = {...message("2", "Bo", "de"), text: "we do", replyTo: quoteOf(question)}

        it("shows who it answers and what they said, in a bubble above it", () => {
            const {container} = render(<ChatLog loading={false} messages={[question, answer]}/>)

            const [, reply] = [...container.querySelectorAll<HTMLElement>(".chat-message")]
            const quote = reply.querySelector<HTMLElement>(".chat-quote")!
            expect(quote.textContent).toBe("In reply to Ana who holds the north")
            expect(quote.style.getPropertyValue("--author-hue")).toBe("165")
            expect(quote.compareDocumentPosition(reply.querySelector(".chat-message-text")!)
                & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
            expect(container.querySelectorAll(".chat-quote")).toHaveLength(1)
        })

        it("paints the quoted name in its own author's color, whatever the color of who answers", () => {
            const guest = {...message("3", "guest_Bo", "de"), text: "nobody"}
            const {container} = render(<ChatLog loading={false} messages={[
                question,
                {...guest, replyTo: quoteOf(question)},
                {...question, id: "4", text: "sure", replyTo: quoteOf(guest)},
            ]}/>)

            const [ofColored, ofGuest] = [...container.querySelectorAll<HTMLElement>(".chat-quote")]
            expect(ofColored.closest<HTMLElement>(".chat-message")!.style.getPropertyValue("--author-chroma")).toBe("0")
            expect(ofColored.style.getPropertyValue("--author-hue")).toBe("165")
            expect(ofColored.style.getPropertyValue("--author-chroma")).toBe("1")
            expect(ofGuest.style.getPropertyValue("--author-chroma")).toBe("0")
        })

        it("quotes only the beginning of a long message", () => {
            const long = {...question, text: "north ".repeat(40).trim()}
            const {container} = render(<ChatLog loading={false} messages={[long, {...answer, replyTo: quoteOf(long)}]}/>)

            const quoted = container.querySelector(".chat-quote-text")!.textContent!
            expect([...quoted]).toHaveLength(QUOTE_MAX_LENGTH + 1)
            expect(quoted.endsWith("…")).toBe(true)
            expect(long.text.startsWith(quoted.slice(0, -1))).toBe(true)
        })

        it("quotes only the message it answers, not what that one answered", () => {
            const followUp = {...message("3", "Cy", "it"), text: "since when", replyTo: quoteOf(answer)}
            const {container} = render(<ChatLog loading={false} messages={[question, answer, followUp]}/>)

            const last = [...container.querySelectorAll<HTMLElement>(".chat-message")].at(-1)!
            expect(last.querySelectorAll(".chat-quote")).toHaveLength(1)
            expect(last.querySelector(".chat-quote")!.textContent).toBe("In reply to Bo we do")
        })

        it("says again who is talking, even in the middle of its author's run", () => {
            const before = {...message("2", "Bo", "de"), text: "hm"}
            const reply = {...answer, id: "3"}
            const {container} = render(<ChatLog loading={false} messages={[question, before, reply]}/>)

            const [, , last] = [...container.querySelectorAll<HTMLElement>(".chat-message")]
            expect(last.querySelector(".chat-message-head")).not.toBeNull()
        })

        it("goes to the message it answers from the quote, and lights it up", async () => {
            const {container} = render(<ChatLog loading={false} messages={[question, answer]}/>)
            const [asked, reply] = [...container.querySelectorAll<HTMLElement>(".chat-message")]
            asked.scrollIntoView = vi.fn()

            await userEvent.click(within(reply).getByRole("button", {name: /^In reply to Ana/}))

            expect(asked.scrollIntoView).toHaveBeenCalledWith({block: "center"})
            expect(asked.classList.contains("chat-message-spotlit")).toBe(true)
            expect(reply.classList.contains("chat-message-spotlit")).toBe(false)
        })

        it("still quotes a message the log no longer holds, with nowhere to go", () => {
            const {container} = render(<ChatLog loading={false} messages={[answer]}/>)

            expect(container.querySelector(".chat-quote")!.textContent).toBe("In reply to Ana who holds the north")
            expect(screen.queryByRole("button", {name: /^In reply to/})).toBeNull()
        })

        it("is started from the button beside a message", async () => {
            const onReply = vi.fn()
            render(<ChatLog loading={false} onReply={onReply} messages={[question, answer]}/>)

            await userEvent.click(screen.getByRole("button", {name: "Reply to Bo"}))

            expect(onReply).toHaveBeenCalledWith(answer)
        })

        it("offers no reply where nothing takes one", () => {
            render(<ChatLog loading={false} messages={[question]}/>)

            expect(screen.queryByRole("button", {name: /^Reply to/})).toBeNull()
        })
    })
})
