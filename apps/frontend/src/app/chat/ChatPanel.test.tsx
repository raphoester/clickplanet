// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen, waitFor} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import ChatPanel from "./ChatPanel.tsx"
import {CHAT_IDENTITY_STORAGE_KEY} from "./chatIdentity.ts"
import {ChatBackend, ChatMessage, ChatRateLimitedError, ChatUnavailableError} from "../../backends/chat.ts"
import {Countries} from "../../domain/countries.ts"

const france = Countries.get("fr")!

const message = (id: string, text: string, sentAt = 1_700_000_000_000): ChatMessage => ({
    id,
    sentAt,
    authorName: "Ana",
    authorTag: "4f2ca1",
    countryCode: "fr",
    text,
})

function stubBackend(history: ChatMessage[] = []) {
    const listeners: ((message: ChatMessage) => void)[] = []

    const backend = {
        getHistory: vi.fn().mockResolvedValue(history),
        listenForMessages: vi.fn((callback: (message: ChatMessage) => void) => {
            listeners.push(callback)
            return () => {
            }
        }),
        sendMessage: vi.fn(async (outgoing) => ({
            id: `sent-${outgoing.text}`,
            sentAt: 1_700_000_100_000,
            authorName: outgoing.authorName,
            authorTag: "c0ffee",
            countryCode: outgoing.countryCode,
            text: outgoing.text,
        })),
    }

    return {
        backend: backend as unknown as ChatBackend & typeof backend,
        broadcast: (m: ChatMessage) => listeners.forEach(listener => listener(m)),
    }
}

const setup = (backend?: ChatBackend) => ({
    ...render(<ChatPanel backend={backend} country={france}/>),
    user: userEvent.setup(),
})

const panel = () => screen.queryByRole("region", {name: "Live chat"})
const item = (text: string) => screen.getByText(text).closest("li")!
const messageBox = () => screen.getByRole("textbox", {name: "Message"})
const nameBox = () => screen.getByLabelText("Pick a name to chat")
const named = async (user: ReturnType<typeof userEvent.setup>, name: string) => {
    await user.type(nameBox(), name)
    await user.click(screen.getByRole("button", {name: "OK"}))
}

beforeEach(() => window.localStorage.clear())
afterEach(cleanup)

describe("ChatPanel", () => {
    it("shows what the server already holds", async () => {
        const {backend} = stubBackend([message("a", "who took Brittany")])
        setup(backend)

        expect(await screen.findByText("who took Brittany")).toBeDefined()
    })

    it("shows a message that arrives on the socket", async () => {
        const {backend, broadcast} = stubBackend()
        setup(backend)

        await screen.findByText("Nobody has said anything yet. Go on.")
        broadcast(message("live", "gm everyone"))

        expect(await screen.findByText("gm everyone")).toBeDefined()
    })

    it("shows a message once when the socket echoes what the send returned", async () => {
        const {backend, broadcast} = stubBackend()
        const {user} = setup(backend)
        await named(user, "Bo")

        await user.type(messageBox(), "hello")
        await user.click(screen.getByRole("button", {name: "Send"}))
        await screen.findByText("hello")

        broadcast({...message("sent-hello", "hello"), authorName: "Bo"})

        await waitFor(() => expect(screen.getAllByText("hello")).toHaveLength(1))
    })

    it("renders message text as text, never as markup", async () => {
        const {backend} = stubBackend([message("a", "<img src=x onerror=alert(1)>")])
        const {container} = setup(backend)

        expect(await screen.findByText("<img src=x onerror=alert(1)>")).toBeDefined()
        expect(container.querySelector("img")).toBeNull()
    })

    describe("showing that a message just landed", () => {
        it("highlights one that arrives while the panel is open", async () => {
            const {backend, broadcast} = stubBackend([message("a", "old news")])
            setup(backend)
            await screen.findByText("old news")

            broadcast(message("live", "gm everyone", 1_700_000_200_000))

            await waitFor(() => expect(item("gm everyone").className).toContain("chat-message-new"))
            expect(item("old news").className).not.toContain("chat-message-new")
        })

        it("stops highlighting it once it has been seen", async () => {
            vi.useFakeTimers()
            try {
                const {backend, broadcast} = stubBackend([message("a", "old news")])
                render(<ChatPanel backend={backend} country={france}/>)
                await vi.waitFor(() => item("old news"))

                broadcast(message("live", "gm everyone", 1_700_000_200_000))
                await vi.waitFor(() => item("gm everyone"))
                expect(item("gm everyone").className).toContain("chat-message-new")

                await act(async () => void vi.advanceTimersByTime(5_000))

                expect(item("gm everyone").className).not.toContain("chat-message-new")
            } finally {
                vi.useRealTimers()
            }
        })

        it("does not highlight the message you just sent yourself", async () => {
            window.localStorage.setItem(
                CHAT_IDENTITY_STORAGE_KEY,
                JSON.stringify({authorId: "author-1", name: "Ana"}),
            )
            const {backend} = stubBackend()
            const {user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await user.type(messageBox(), "mine{Enter}")
            await screen.findByText("mine")

            expect(item("mine").className).not.toContain("chat-message-new")
        })

        it("gives every author their own colour, and the same one every time", async () => {
            const {backend} = stubBackend([
                {...message("a", "first"), authorName: "Ana", authorTag: "4f2ca1"},
                {...message("b", "second", 1_700_000_100_000), authorName: "Bo", authorTag: "c0ffee"},
                {...message("c", "third", 1_700_000_200_000), authorName: "Ana", authorTag: "4f2ca1"},
            ])
            setup(backend)
            await screen.findByText("first")

            const hue = (text: string) => item(text).style.getPropertyValue("--author-hue")

            expect(hue("first")).not.toBe("")
            expect(hue("third")).toBe(hue("first"))
            expect(hue("second")).not.toBe(hue("first"))
        })
    })

    describe("the name", () => {
        it("is asked for before the first message", async () => {
            const {backend} = stubBackend()
            setup(backend)

            expect(await screen.findByLabelText("Pick a name to chat")).toBeDefined()
            expect(screen.queryByRole("textbox", {name: "Message"})).toBeNull()
        })

        it("is refused while it is blank or too long", async () => {
            const {backend} = stubBackend()
            const {user} = setup(backend)
            await screen.findByLabelText("Pick a name to chat")

            const ok = () => screen.getByRole("button", {name: "OK"})
            expect(ok()).toHaveProperty("disabled", true)

            await user.type(nameBox(), "x".repeat(25))
            expect(ok()).toHaveProperty("disabled", true)
        })

        it("opens the composer once it is picked, and is kept for next time", async () => {
            const {backend} = stubBackend()
            const {user} = setup(backend)
            await screen.findByLabelText("Pick a name to chat")

            await named(user, "Ana")

            expect(messageBox()).toBeDefined()
            await waitFor(() => expect(window.localStorage.getItem(CHAT_IDENTITY_STORAGE_KEY))
                .toContain("Ana"))
        })

        it("is not asked for again on a later visit", async () => {
            window.localStorage.setItem(
                CHAT_IDENTITY_STORAGE_KEY,
                JSON.stringify({authorId: "author-1", name: "Ana"}),
            )
            const {backend} = stubBackend()
            setup(backend)

            expect(await screen.findByRole("textbox", {name: "Message"})).toBeDefined()
        })

        it("can be changed", async () => {
            window.localStorage.setItem(
                CHAT_IDENTITY_STORAGE_KEY,
                JSON.stringify({authorId: "author-1", name: "Ana"}),
            )
            const {backend} = stubBackend()
            const {user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await user.click(screen.getByRole("button", {name: "Change"}))
            expect(nameBox()).toBeDefined()
        })
    })

    describe("sending", () => {
        beforeEach(() => window.localStorage.setItem(
            CHAT_IDENTITY_STORAGE_KEY,
            JSON.stringify({authorId: "author-1", name: "Ana"}),
        ))

        it("posts the message with the identity and the country being played", async () => {
            const {backend} = stubBackend()
            const {user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await user.type(messageBox(), "hello{Enter}")

            expect(backend.sendMessage).toHaveBeenCalledWith({
                authorName: "Ana",
                authorId: "author-1",
                countryCode: "fr",
                text: "hello",
            })
        })

        it("clears the box on success", async () => {
            const {backend} = stubBackend()
            const {user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await user.type(messageBox(), "hello{Enter}")

            await waitFor(() => expect(messageBox()).toHaveProperty("value", ""))
        })

        it("keeps what was typed while the message was in flight", async () => {
            const {backend} = stubBackend()
            let land = (message: ChatMessage) => void message
            backend.sendMessage.mockImplementation(() => new Promise(resolve => {
                land = resolve as typeof land
            }))
            const {user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await user.type(messageBox(), "first{Enter}")
            await user.type(messageBox(), "second")
            land(message("first", "first"))

            await waitFor(() => expect(messageBox()).toHaveProperty("value", "second"))
        })

        it("keeps the text when the server refuses it, and says why", async () => {
            const {backend} = stubBackend()
            backend.sendMessage.mockRejectedValue(new ChatRateLimitedError())
            vi.spyOn(console, "error").mockImplementation(() => {})
            const {user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await user.type(messageBox(), "hello{Enter}")

            expect(await screen.findByRole("alert")).toHaveProperty(
                "textContent",
                "You're sending messages too fast. Give it a few seconds.",
            )
            expect(messageBox()).toHaveProperty("value", "hello")
        })

        it("refuses an empty message without asking the server", async () => {
            const {backend} = stubBackend()
            const {user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await user.type(messageBox(), "   {Enter}")

            expect(backend.sendMessage).not.toHaveBeenCalled()
        })

        it("refuses a message the server would reject as too long", async () => {
            const {backend} = stubBackend()
            const {user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await user.click(messageBox())
            await user.paste("x".repeat(281))
            await user.click(screen.getByRole("button", {name: "Send"}))

            expect(backend.sendMessage).not.toHaveBeenCalled()
        })
    })

    describe("when the server has no chat", () => {
        it("shows nothing at all rather than an empty box", async () => {
            const {backend} = stubBackend()
            backend.getHistory.mockRejectedValue(new ChatUnavailableError())
            vi.spyOn(console, "error").mockImplementation(() => {})
            setup(backend)

            await waitFor(() => expect(panel()).toBeNull())
        })

        it("shows nothing when no chat backend was wired at all", () => {
            setup(undefined)

            expect(panel()).toBeNull()
        })
    })

    describe("folded", () => {
        beforeEach(() => vi.stubGlobal("matchMedia", () => ({matches: true})))
        afterEach(() => vi.unstubAllGlobals())

        it("starts closed on a small screen and opens on a click", async () => {
            const {backend} = stubBackend([message("a", "who took Brittany")])
            const {user} = setup(backend)

            expect(screen.queryByText("who took Brittany")).toBeNull()

            await user.click(screen.getByRole("button", {name: /Chat/}))

            expect(await screen.findByText("who took Brittany")).toBeDefined()
        })

        it("quotes the newest message under the header while it is closed", async () => {
            const {backend, broadcast} = stubBackend([message("a", "seen already")])
            setup(backend)
            await waitFor(() => expect(backend.getHistory).toHaveBeenCalled())

            broadcast({...message("b", "who took Brittany", 1_700_000_200_000), authorName: "Ana"})

            const header = await screen.findByRole("button", {name: /Chat/})
            await waitFor(() => expect(header.textContent).toContain("who took Brittany"))
            expect(header.textContent).toContain("Ana")
        })

        it("highlights what was missed once it is opened", async () => {
            const {backend, broadcast} = stubBackend([message("a", "seen already")])
            const {user} = setup(backend)
            await waitFor(() => expect(backend.getHistory).toHaveBeenCalled())

            broadcast(message("b", "and one more", 1_700_000_200_000))
            await screen.findByLabelText("1 new message")

            await user.click(screen.getByRole("button", {name: /Chat/}))

            await waitFor(() => expect(item("and one more").className).toContain("chat-message-new"))
            expect(item("seen already").className).not.toContain("chat-message-new")
        })

        it("counts what arrived while it was closed, and only that", async () => {
            const {backend, broadcast} = stubBackend([message("a", "seen already")])
            setup(backend)
            await waitFor(() => expect(backend.getHistory).toHaveBeenCalled())

            expect(screen.queryByLabelText(/new messages/)).toBeNull()

            broadcast(message("b", "and one more", 1_700_000_200_000))

            expect(await screen.findByLabelText("1 new message")).toBeDefined()
        })
    })
})
