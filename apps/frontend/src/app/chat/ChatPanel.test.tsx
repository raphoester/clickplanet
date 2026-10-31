// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import ChatPanel from "./ChatPanel.tsx"
import {CHAT_IDENTITY_STORAGE_KEY} from "./chatIdentity.ts"
import {CHAT_SIZE_STORAGE_KEY} from "./chatSize.ts"
import {WANTED_HEIGHT, WANTED_WIDTH} from "./useChatSize.ts"
import {
    ChatAnnouncement,
    ChatBackend,
    ChatMessage,
    ChatMessageGoneError,
    ChatMutedError,
    ChatNoSessionError,
    ChatRateLimitedError,
    ChatRejectedError,
    OutgoingReaction,
    Reaction,
    ReactionsChange,
} from "../../backends/chat.ts"
import {Countries} from "../../domain/countries.ts"
import {NameColor} from "../../backends/player.ts"

const france = Countries.get("fr")!

const OWN_GUEST = "guest_c0ffee"

const OWN_ACCOUNT = "account-own"

const message = (id: string, text: string, sentAt = 1_700_000_000_000): ChatMessage => ({
    id,
    sentAt,
    authorAccountId: "account-ana",
    authorName: "Ana",
    authorAdmin: false,
    authorColor: NameColor.UNSPECIFIED,
    authorStreak: 0,
    countryCode: "fr",
    text,
    reactions: [],
    reactionsVersion: 0,
})

function stubBackend(history: ChatMessage[] = [], announcements: ChatAnnouncement[] = [], seenUntil?: number) {
    const listeners: ((message: ChatMessage) => void)[] = []
    const reactionListeners: ((change: ReactionsChange) => void)[] = []
    const announcementListeners: ((announcement: ChatAnnouncement) => void)[] = []
    const removalListeners: ((messageId: string) => void)[] = []

    const backend = {
        getHistory: vi.fn().mockResolvedValue({messages: history, announcements, seenUntil}),
        markSeen: vi.fn().mockResolvedValue(undefined),
        listenForMessages: vi.fn((
            callback: (message: ChatMessage) => void,
            onReactions?: (change: ReactionsChange) => void,
            onAnnouncement?: (announcement: ChatAnnouncement) => void,
            _onResumed?: () => void,
            onRemoved?: (messageId: string) => void,
        ) => {
            listeners.push(callback)
            if (onReactions) reactionListeners.push(onReactions)
            if (onAnnouncement) announcementListeners.push(onAnnouncement)
            if (onRemoved) removalListeners.push(onRemoved)
            return () => {
            }
        }),
        react: vi.fn(async (outgoing: OutgoingReaction): Promise<ReactionsChange> => ({
            messageId: outgoing.messageId,
            reactions: [{
                reaction: outgoing.reaction,
                count: outgoing.on ? 1 : 0,
                mine: outgoing.on,
                reactors: outgoing.on ? [OWN_GUEST] : [],
            }].filter(count => count.count > 0),
            version: 1,
        })),
        sendMessage: vi.fn(async (outgoing) => ({
            id: `sent-${outgoing.text}`,
            sentAt: 1_700_000_100_000,
            authorAccountId: OWN_ACCOUNT,
            authorName: OWN_GUEST,
            countryCode: outgoing.countryCode,
            text: outgoing.text,
            reactions: [] as ChatMessage["reactions"],
            reactionsVersion: 0,
        })),
    }

    return {
        backend: backend as unknown as ChatBackend & typeof backend,
        broadcast: (m: ChatMessage) => listeners.forEach(listener => listener(m)),
        broadcastReactions: (change: ReactionsChange) => reactionListeners.forEach(listener => listener(change)),
        announce: (announcement: ChatAnnouncement) => announcementListeners.forEach(listener => listener(announcement)),
        remove: (messageId: string) => removalListeners.forEach(listener => listener(messageId)),
    }
}

const setup = (backend?: ChatBackend, username?: string, accountId?: string) => ({
    ...render(<ChatPanel backend={backend} country={france} username={username} accountId={accountId}/>),
    user: userEvent.setup(),
})

const panel = () => screen.queryByRole("region", {name: "Live chat"})
const item = (text: string) => screen.getByText(text).closest("li")!
const messageBox = () => screen.getByRole("textbox", {name: "Message"})
const send = async (user: ReturnType<typeof userEvent.setup>, text: string) => {
    await user.type(messageBox(), text)
    await user.click(screen.getByRole("button", {name: "Send"}))
}

beforeEach(() => window.localStorage.clear())
afterEach(cleanup)

describe("ChatPanel sound", () => {
    const withSound = (backend: ChatBackend, username?: string, accountId?: string) => {
        const playSound = vi.fn()
        return {
            playSound,
            user: userEvent.setup(),
            ...render(<ChatPanel backend={backend}
                                 country={france}
                                 playSound={playSound}
                                 username={username}
                                 accountId={accountId}/>),
        }
    }

    it("pings for someone else's message, not for the history it opens on", async () => {
        const {backend, broadcast} = stubBackend([message("old", "from before")])
        const {playSound} = withSound(backend)

        await screen.findByText("from before")
        expect(playSound).not.toHaveBeenCalled()

        broadcast(message("live", "gm everyone", 1_700_000_050_000))
        await waitFor(() => expect(playSound).toHaveBeenCalledWith("chat"))
    })

    it("stays quiet for your own message, once its answer is in", async () => {
        const {backend, broadcast} = stubBackend()
        const {playSound, user} = withSound(backend)
        await screen.findByText("Nobody has said anything yet. Go on.")

        await send(user, "hello")
        await screen.findByText("hello")
        broadcast({...message("sent-hello", "hello"), authorName: OWN_GUEST})

        expect(playSound).not.toHaveBeenCalled()
    })

    it("stays quiet for a guest's own message whose broadcast comes first, once its account is known", async () => {
        const {backend, broadcast} = stubBackend()
        const {playSound, user} = withSound(backend)
        await screen.findByText("Nobody has said anything yet. Go on.")
        await send(user, "first")
        await screen.findByText("first")

        broadcast({...message("sent-hello", "hello", 1_700_000_200_000), authorAccountId: OWN_ACCOUNT, authorName: OWN_GUEST})
        await screen.findByText("hello")
        await send(user, "hello")

        expect(playSound).not.toHaveBeenCalled()
    })

    it("stays quiet for your own message sent from another device", async () => {
        const {backend, broadcast} = stubBackend([message("old", "from before")])
        const {playSound} = withSound(backend, "ana_1", OWN_ACCOUNT)
        await screen.findByText("from before")

        broadcast({...message("live", "hello", 1_700_000_050_000), authorAccountId: OWN_ACCOUNT, authorName: "ana_1"})
        await screen.findByText("hello")

        expect(playSound).not.toHaveBeenCalled()
    })

    it("pings for another account, even under the name this player holds", async () => {
        const {backend, broadcast} = stubBackend([message("old", "from before")])
        const {playSound} = withSound(backend, "ana_1", OWN_ACCOUNT)
        await screen.findByText("from before")

        broadcast({...message("live", "hello", 1_700_000_050_000), authorAccountId: "account-other", authorName: "ana_1"})

        await waitFor(() => expect(playSound).toHaveBeenCalledWith("chat"))
    })

    it("pings for a message with no account, even under the name this player holds", async () => {
        const {backend, broadcast} = stubBackend([message("old", "from before")])
        const {playSound} = withSound(backend, "ana_1")
        await screen.findByText("from before")

        broadcast({...message("live", "hello", 1_700_000_050_000), authorAccountId: "", authorName: "ana_1"})

        await waitFor(() => expect(playSound).toHaveBeenCalledWith("chat"))
    })

    it("pings for another guest", async () => {
        const {backend, broadcast} = stubBackend([message("old", "from before")])
        const {playSound, user} = withSound(backend)
        await screen.findByText("from before")
        await send(user, "mine")
        await screen.findByText("mine")

        broadcast({...message("live", "hello", 1_700_000_200_000), authorName: "guest_91aa3d"})

        await waitFor(() => expect(playSound).toHaveBeenCalledWith("chat"))
    })
})

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
        await screen.findByRole("textbox", {name: "Message"})

        await send(user, "hello")
        await screen.findByText("hello")

        broadcast({...message("sent-hello", "hello"), authorName: OWN_GUEST})

        await waitFor(() => expect(screen.getAllByText("hello")).toHaveLength(1))
    })

    it("shows a bomb from the history as a line between the messages, not a bubble", async () => {
        const bomb: ChatAnnouncement = {
            kind: "bomb", id: "boom", announcedAt: 1_700_000_000_500, country: "fr", ground: "de", tile: 42, cleared: 3,
        }
        const {backend} = stubBackend([message("a", "before"), message("b", "after", 1_700_000_001_000)], [bomb])
        setup(backend)

        const line = (await screen.findByText("bombed Germany")).closest("li")!
        expect(line.className).toBe("chat-announcement")
        expect(line.textContent).toContain("France")
        expect(line.previousElementSibling?.textContent).toContain("before")
        expect(line.nextElementSibling?.textContent).toContain("after")
        expect(item("after").className).toContain("chat-message-opens")
    })

    it("names the landmass a bomb hit, not the country it belongs to", async () => {
        const bomb: ChatAnnouncement = {
            kind: "bomb", id: "boom", announcedAt: 1_700_000_000_500, country: "fr", ground: "au", landmass: 420, tile: 42, cleared: 3,
        }
        const {backend} = stubBackend([message("a", "before")], [bomb])
        setup(backend)

        expect(await screen.findByText("bombed Tasmania")).toBeTruthy()
    })

    it("says who was muted and for how long, as a line between the messages", async () => {
        const mute: ChatAnnouncement = {kind: "mute", id: "hush", announcedAt: 1_700_000_000_500, name: "guest_a1b2c3", seconds: 3600}
        const {backend} = stubBackend([message("a", "We target the players"), message("b", "thanks", 1_700_000_001_000)], [mute])
        setup(backend)

        const line = (await screen.findByText("has been muted for one hour", {exact: false})).closest("li")!
        expect(line.className).toBe("chat-announcement")
        expect(line.textContent).toContain("guest_a1b2c3 has been muted for one hour")
        expect(line.previousElementSibling?.textContent).toContain("We target the players")
    })

    it("shows a mute announced while the chat is open", async () => {
        const {backend, announce} = stubBackend()
        setup(backend)

        await screen.findByText("Nobody has said anything yet. Go on.")
        act(() => announce({kind: "mute", id: "hush", announcedAt: Date.now(), name: "Ada_L", seconds: 7200}))

        expect(await screen.findByText("has been muted for 2 hours", {exact: false})).toBeDefined()
    })

    it("shows a bomb that lands while the chat is open", async () => {
        const {backend, announce} = stubBackend()
        setup(backend)

        await screen.findByText("Nobody has said anything yet. Go on.")
        act(() => announce({kind: "bomb", id: "splash", announcedAt: Date.now(), country: "fr", cleared: 0}))

        expect(await screen.findByText("bombed the ocean")).toBeDefined()
    })

    it("says which flag fortified which territory, by the territory's own name", async () => {
        const {backend, announce} = stubBackend()
        setup(backend)

        await screen.findByText("Nobody has said anything yet. Go on.")
        act(() => announce({kind: "fortify", id: "wall", announcedAt: Date.now(), country: "es", ground: "fr", landmass: 359, tiles: 137}))

        const line = (await screen.findByText("fortified French Guiana", {exact: false})).closest("li")!
        expect(line.className).toBe("chat-announcement")
        expect(line.textContent).toContain("Spain fortified French Guiana")
    })

    it("says which country won the season, with its flag", async () => {
        const {backend, announce} = stubBackend()
        setup(backend)

        await screen.findByText("Nobody has said anything yet. Go on.")
        act(() => announce({kind: "champion", id: "crown", announcedAt: Date.now(), season: 0, countries: ["de"]}))

        const line = (await screen.findByText("Germany wins Season 0")).closest("li")!
        expect(line.className).toContain("chat-announcement-champion")
        expect(within(line).getByText("🏆")).toBeDefined()
        expect(line.querySelectorAll(".country-flag")).toHaveLength(1)
    })

    it("tells a day's result by its number, and a battle's as the battle's, with its times", async () => {
        const {backend, announce} = stubBackend()
        setup(backend)

        await screen.findByText("Nobody has said anything yet. Go on.")
        const podium = [{country: "fr", rank: 1, points: 75}]
        act(() => announce({kind: "round", id: "day", announcedAt: Date.now(), number: 1, finale: false, own: false, times: 1, podium}))
        act(() => announce({kind: "round", id: "battle", announcedAt: Date.now(), number: 1, finale: false, own: true, times: 3, podium}))
        act(() => announce({kind: "round", id: "finale", announcedAt: Date.now(), number: 2, finale: true, own: true, times: 3, podium}))

        expect(await screen.findByText("Day 1 is over!")).toBeDefined()
        const battle = screen.getByText("The battle is over!").closest("li")!
        expect(within(battle).getByText("×3")).toBeDefined()
        expect(within(battle).queryByText(/Day/)).toBeNull()
        expect(screen.getByText("The Final Battle is over!")).toBeDefined()
    })

    it("says a message an operator removed was deleted, in its place and in the quote of a reply to it", async () => {
        const insult = message("a", "you all suck")
        const {backend, remove} = stubBackend([
            insult,
            {...message("b", "say that again", 1_700_000_100_000), authorAccountId: "account-bo", authorName: "Bo", replyTo: {
                messageId: "a", authorName: "Ana", authorColor: NameColor.UNSPECIFIED, text: "you all suck",
            }},
        ])
        setup(backend)

        expect(await screen.findAllByText("you all suck")).toHaveLength(2)
        act(() => remove("a"))

        expect(screen.queryByText("you all suck")).toBeNull()
        expect(screen.getAllByText("This message has been deleted")).toHaveLength(2)
        expect(screen.getByText("say that again")).toBeDefined()
        expect(screen.queryByRole("button", {name: "Reply to Ana"})).toBeNull()
        expect(screen.queryByRole("button", {name: "Report this message from Ana"})).toBeNull()
    })

    it("renders message text as text, never as markup", async () => {
        const {backend} = stubBackend([message("a", "<img src=x onerror=alert(1)>")])
        const {container} = setup(backend)

        expect(await screen.findByText("<img src=x onerror=alert(1)>")).toBeDefined()
        expect(container.querySelector("img")).toBeNull()
    })

    it("marks a message with its author's flag, not with a country code", async () => {
        const {backend} = stubBackend([message("a", "Bonjour")])
        const {container} = setup(backend)

        await screen.findByText("Bonjour")
        const badge = container.querySelector(".chat-message-country")!
        expect(badge.querySelectorAll(".country-flag")).toHaveLength(1)
        expect(badge.textContent).toBe("")
        expect(badge.getAttribute("title")).toBe("France")
    })

    it("names the country its flag stands for, for anyone who cannot see it", async () => {
        const {backend} = stubBackend([message("a", "Bonjour")])
        setup(backend)

        await screen.findByText("Bonjour")
        expect(screen.getByRole("img", {name: "France"})).toBeDefined()
    })

    it("introduces an author once, not on every message of their run", async () => {
        const {backend} = stubBackend([
            message("a", "first"),
            message("b", "second", 1_700_000_060_000),
            {...message("c", "third", 1_700_000_120_000), authorName: "Bo"},
        ])
        const {container} = setup(backend)
        await screen.findByText("first")

        expect(container.querySelectorAll(".chat-message-head")).toHaveLength(2)
        expect(item("first").querySelector(".chat-message-head")).not.toBeNull()
        expect(item("second").querySelector(".chat-message-head")).toBeNull()
        expect(item("third").querySelector(".chat-message-head")).not.toBeNull()
    })

    it("opens a new group when the same author speaks again much later", async () => {
        const {backend} = stubBackend([
            message("a", "first"),
            message("b", "much later", 1_700_000_000_000 + 10 * 60_000),
        ])
        const {container} = setup(backend)
        await screen.findByText("first")

        expect(container.querySelectorAll(".chat-message-head")).toHaveLength(2)
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
            const {backend} = stubBackend()
            const {user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await user.type(messageBox(), "mine{Enter}")
            await screen.findByText("mine")

            expect(item("mine").className).not.toContain("chat-message-new")
        })

        it("paints every message in the colour its author chose", async () => {
            const {backend} = stubBackend([
                {...message("a", "first"), authorName: "Ana", authorColor: NameColor.TEAL},
                {...message("b", "second", 1_700_000_100_000), authorName: "Bo", authorColor: NameColor.PINK},
                {...message("c", "third", 1_700_000_200_000), authorName: "Ana", authorColor: NameColor.TEAL},
            ])
            setup(backend)
            await screen.findByText("first")

            const hue = (text: string) => item(text).style.getPropertyValue("--author-hue")

            expect(hue("first")).toBe("165")
            expect(hue("second")).toBe("330")
            expect(hue("third")).toBe("165")
        })
    })

    describe("as a guest", () => {
        it("asks for no name: the composer is open at once", async () => {
            const {backend} = stubBackend()
            const {container} = setup(backend)

            expect(await screen.findByRole("textbox", {name: "Message"})).toBeDefined()
            expect(screen.queryByRole("button", {name: "Change"})).toBeNull()
            expect(container.querySelector(".chat-identity")?.textContent).toBe("as a guest")
        })

        it("says the name the server gave it once a message went out", async () => {
            const {backend} = stubBackend()
            const {container, user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await send(user, "hello")

            await waitFor(() => expect(container.querySelector(".chat-identity")?.textContent)
                .toBe(`as ${OWN_GUEST}`))
        })

        it("keeps its author id for next time, and drops a name an older build stored", async () => {
            window.localStorage.setItem(
                CHAT_IDENTITY_STORAGE_KEY,
                JSON.stringify({authorId: "author-1", name: "Ana"}),
            )
            const {backend} = stubBackend()
            setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await waitFor(() => expect(JSON.parse(window.localStorage.getItem(CHAT_IDENTITY_STORAGE_KEY)!))
                .toEqual({authorId: "author-1"}))
        })
    })

    describe("with a username", () => {
        it("asks for no name, and says the username it posts as", async () => {
            const {backend} = stubBackend()
            const {container} = setup(backend, "ana_1")

            expect(await screen.findByRole("textbox", {name: "Message"})).toBeDefined()
            expect(container.querySelector(".chat-identity")?.textContent).toBe("as ana_1")
            expect(screen.queryByRole("button", {name: "Change"})).toBeNull()
        })

        it("sends no name: the server names the sender", async () => {
            const {backend} = stubBackend()
            const {user} = setup(backend, "ana_1")
            await screen.findByRole("textbox", {name: "Message"})

            await user.type(messageBox(), "hello{Enter}")

            expect(backend.sendMessage.mock.calls[0][0]).not.toHaveProperty("authorName")
        })
    })

    describe("sending", () => {
        beforeEach(() => window.localStorage.setItem(
            CHAT_IDENTITY_STORAGE_KEY,
            JSON.stringify({authorId: "author-1"}),
        ))

        it("posts the message with the identity and the country being played", async () => {
            const {backend} = stubBackend()
            const {user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await user.type(messageBox(), "hello{Enter}")

            expect(backend.sendMessage).toHaveBeenCalledWith({
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

        it("says until when a muted player is muted, and keeps the text", async () => {
            const until = Date.now() + 3600_000
            const {backend} = stubBackend()
            backend.sendMessage.mockRejectedValue(new ChatMutedError(until))
            vi.spyOn(console, "error").mockImplementation(() => {})
            const {user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await user.type(messageBox(), "hello{Enter}")

            const clock = new Intl.DateTimeFormat(undefined, {hour: "2-digit", minute: "2-digit"})
            expect(await screen.findByRole("alert")).toHaveProperty("textContent", `You are muted until ${clock.format(until)}.`)
            expect(messageBox()).toHaveProperty("value", "hello")
        })

        it("says so when no session could be had to send it", async () => {
            const {backend} = stubBackend()
            backend.sendMessage.mockRejectedValue(new ChatNoSessionError())
            vi.spyOn(console, "error").mockImplementation(() => {})
            const {user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await user.type(messageBox(), "hello{Enter}")

            expect(await screen.findByRole("alert")).toHaveProperty(
                "textContent",
                "Could not start a session to chat. Try again in a moment.",
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

        it("keeps a pasted message on one line", async () => {
            const {backend} = stubBackend()
            const {user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await user.click(messageBox())
            await user.paste("first line\nsecond line")

            expect(messageBox()).toHaveProperty("value", "first line second line")
        })

        it("sends on Shift+Enter rather than breaking the line", async () => {
            const {backend} = stubBackend()
            const {user} = setup(backend)
            await screen.findByRole("textbox", {name: "Message"})

            await user.type(messageBox(), "hello{Shift>}{Enter}{/Shift}")

            await waitFor(() => expect(backend.sendMessage).toHaveBeenCalledTimes(1))
            expect(messageBox()).toHaveProperty("value", "")
        })
    })

    describe("when the chat cannot be loaded", () => {
        it("shows nothing at all rather than an empty box", async () => {
            const {backend} = stubBackend()
            backend.getHistory.mockRejectedValue(new Error("boom"))
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

describe("ChatPanel's seen mark", () => {
    const T = 1_700_000_000_000
    const bomb = (id: string, announcedAt: number): ChatAnnouncement =>
        ({kind: "bomb", id, announcedAt, country: "de", cleared: 3})

    afterEach(() => {
        vi.useRealTimers()
        vi.unstubAllGlobals()
    })

    describe("folded", () => {
        beforeEach(() => vi.stubGlobal("matchMedia", () => ({matches: true})))

        it("counts every line said since the last visit, bombs too", async () => {
            const {backend} = stubBackend(
                [message("a", "seen last time", T), message("b", "said while away", T + 2_000)],
                [bomb("x", T + 1_000)],
                T)
            setup(backend)

            expect(await screen.findByLabelText("2 new messages")).toBeDefined()
        })

        it("does not count what the player said itself on another visit", async () => {
            const {backend} = stubBackend([message("a", "mine, from my phone", T + 1_000)], [], T)
            setup(backend, "Ana", "account-ana")
            await waitFor(() => expect(backend.getHistory).toHaveBeenCalled())

            expect(screen.queryByLabelText(/new message/)).toBeNull()
        })

        it("counts what another account said under the name the player holds", async () => {
            const {backend} = stubBackend([message("a", "not mine", T + 1_000)], [], T)
            setup(backend, "Ana", OWN_ACCOUNT)

            expect(await screen.findByLabelText("1 new message")).toBeDefined()
        })

        it("counts nothing on a first visit, and keeps a mark for the next one", async () => {
            vi.useFakeTimers({shouldAdvanceTime: true})
            const {backend} = stubBackend([message("a", "before you came", T)])
            setup(backend)
            await act(async () => {
            })

            await act(async () => void vi.advanceTimersByTime(2_000))

            expect(screen.queryByLabelText(/new message/)).toBeNull()
            expect(backend.markSeen).toHaveBeenCalledWith(T)
        })

        it("marks nothing while it stays closed", async () => {
            vi.useFakeTimers({shouldAdvanceTime: true})
            const {backend, broadcast} = stubBackend([message("a", "seen last time", T)], [], T)
            setup(backend)
            await screen.findByLabelText("Live chat")

            broadcast(message("b", "while closed", T + 1_000))
            await screen.findByLabelText("1 new message")
            await act(async () => void vi.advanceTimersByTime(5_000))

            expect(backend.markSeen).not.toHaveBeenCalled()
        })

        it("puts what was missed since the last visit under an Unread line once it is opened", async () => {
            const {backend} = stubBackend([message("a", "seen last time", T), message("b", "said while away", T + 1_000)], [], T)
            const {user} = setup(backend)
            await screen.findByLabelText("1 new message")

            await user.click(screen.getByRole("button", {name: /Chat/}))

            const unread = await screen.findByRole("separator")
            expect(unread.textContent).toBe("Unread")
            expect(item("said while away").previousElementSibling).toBe(unread)
        })
    })

    it("marks what lands while it is open, a bomb too, once things settle", async () => {
        vi.useFakeTimers({shouldAdvanceTime: true})
        const {backend, broadcast, announce} = stubBackend([message("a", "seen last time", T)], [], T)
        setup(backend)
        await vi.waitFor(() => item("seen last time"))

        broadcast(message("b", "one", T + 1_000))
        announce(bomb("x", T + 2_000))
        await vi.waitFor(() => item("one"))
        expect(backend.markSeen).not.toHaveBeenCalled()

        await act(async () => void vi.advanceTimersByTime(2_000))

        expect(backend.markSeen).toHaveBeenCalledTimes(1)
        expect(backend.markSeen).toHaveBeenCalledWith(T + 2_000)
    })

    it("keeps the mark at once when the page goes away", async () => {
        const {backend, broadcast} = stubBackend([message("a", "seen last time", T)], [], T)
        setup(backend)
        await screen.findByText("seen last time")

        broadcast(message("b", "just now", T + 1_000))
        await screen.findByText("just now")
        window.dispatchEvent(new Event("pagehide"))

        expect(backend.markSeen).toHaveBeenCalledWith(T + 1_000)
    })

    it("does not count a hidden page as seeing, even open", async () => {
        const hidden = vi.spyOn(document, "hidden", "get").mockReturnValue(true)
        try {
            const {backend, broadcast} = stubBackend([message("a", "seen last time", T)], [], T)
            const onUnread = vi.fn()
            render(<ChatPanel backend={backend} country={france} onUnread={onUnread}/>)
            await screen.findByText("seen last time")

            broadcast(message("b", "while you were away", T + 1_000))

            await waitFor(() => expect(onUnread).toHaveBeenLastCalledWith(1))
            hidden.mockReturnValue(false)
            act(() => void document.dispatchEvent(new Event("visibilitychange")))
            await waitFor(() => expect(onUnread).toHaveBeenLastCalledWith(0))
        } finally {
            hidden.mockRestore()
        }
    })

    it("tells a phone's tab what was missed since the last visit", async () => {
        const {backend} = stubBackend([message("a", "said while away", T + 1_000)], [bomb("x", T + 2_000)], T)
        const onUnread = vi.fn()
        render(<ChatPanel backend={backend} country={france} compact open={false} onUnread={onUnread}/>)

        await waitFor(() => expect(onUnread).toHaveBeenLastCalledWith(2))
    })
})

describe("ChatPanel's Unread line", () => {
    const T = 1_700_000_000_000
    const LINE_PX = 50
    const VIEW_PX = 150

    const box = (top: number, bottom: number) =>
        ({top, bottom, left: 0, right: 0, width: 0, height: bottom - top, x: 0, y: top, toJSON: () => ({})}) as DOMRect

    // jsdom lays nothing out: every line of the log is LINE_PX tall, in a log VIEW_PX tall.
    function layOut() {
        const lines = (log: Element) => [...log.querySelectorAll(".chat-messages > li")]
        vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (this: Element) {
            if (this.classList.contains("chat-log")) return box(0, VIEW_PX)
            const log = this.closest(".chat-log")
            const index = log ? lines(log).indexOf(this) : -1
            if (!log || index < 0) return box(0, 0)
            const top = index * LINE_PX - log.scrollTop
            return box(top, top + LINE_PX)
        })
        vi.spyOn(Element.prototype, "scrollHeight", "get").mockImplementation(function (this: Element) {
            return lines(this).length * LINE_PX
        })
        vi.spyOn(Element.prototype, "clientHeight", "get").mockImplementation(function (this: Element) {
            return this.classList.contains("chat-log") ? VIEW_PX : 0
        })
    }

    const backlog = () => stubBackend(
        [0, 1, 2, 3, 4, 5].map(i => message(`m${i}`, `line ${i}`, T + i * 1_000)),
        [],
        T + 1_000)

    const open = (backend: ChatBackend, onUnread = vi.fn()) => ({
        onUnread,
        user: userEvent.setup({advanceTimers: vi.advanceTimersByTime}),
        ...render(<ChatPanel backend={backend} country={france} onUnread={onUnread}/>),
    })

    const log = () => document.querySelector(".chat-log")!

    beforeEach(() => {
        vi.useFakeTimers({shouldAdvanceTime: true})
        layOut()
    })

    afterEach(() => {
        vi.useRealTimers()
        vi.restoreAllMocks()
    })

    it("opens on the first unread line, under an Unread line, and marks only what it shows", async () => {
        const {backend} = backlog()
        const {onUnread} = open(backend)

        const unread = await screen.findByRole("separator")
        expect(item("line 2").previousElementSibling).toBe(unread)
        expect(log().scrollTop).toBeGreaterThan(0)
        await waitFor(() => expect(onUnread).toHaveBeenLastCalledWith(3))

        await act(async () => void vi.advanceTimersByTime(2_000))
        expect(backend.markSeen).toHaveBeenCalledTimes(1)
        expect(backend.markSeen).toHaveBeenCalledWith(T + 2_000)
    })

    it("marks more as the player scrolls down, in one call however fast", async () => {
        const {backend} = backlog()
        const {onUnread} = open(backend)
        await screen.findByRole("separator")

        for (const top of [100, 150, 176]) {
            log().scrollTop = top
            fireEvent.scroll(log())
        }

        await waitFor(() => expect(onUnread).toHaveBeenLastCalledWith(1))
        await act(async () => void vi.advanceTimersByTime(2_000))
        expect(backend.markSeen).toHaveBeenCalledTimes(1)
        expect(backend.markSeen).toHaveBeenCalledWith(T + 4_000)
    })

    it("goes to the newest line from the button, and so marks everything", async () => {
        const {backend} = backlog()
        const {onUnread, user} = open(backend)
        await screen.findByRole("separator")

        await user.click(screen.getByRole("button", {name: "New messages"}))

        await waitFor(() => expect(onUnread).toHaveBeenLastCalledWith(0))
        expect(screen.queryByRole("button", {name: "New messages"})).toBeNull()
        await act(async () => void vi.advanceTimersByTime(2_000))
        expect(backend.markSeen).toHaveBeenLastCalledWith(T + 5_000)
    })

    it("draws no Unread line when nothing was missed, and opens on the newest", async () => {
        const {backend} = stubBackend([0, 1, 2, 3].map(i => message(`m${i}`, `line ${i}`, T + i * 1_000)), [], T + 3_000)
        open(backend)
        await screen.findByText("line 3")

        expect(screen.queryByRole("separator")).toBeNull()
        expect(log().scrollTop).toBe(4 * LINE_PX)
        expect(screen.queryByRole("button", {name: "New messages"})).toBeNull()
    })
})

describe("ChatPanel reactions", () => {
    const clown = (count: number, mine: boolean, reactors: string[] = []) =>
        ({reaction: Reaction.CLOWN, count, mine, reactors})

    it("puts a reaction on from the picker", async () => {
        const {backend} = stubBackend([message("m1", "gm")])
        const {user} = setup(backend, "Ana")
        await screen.findByText("gm")

        await user.click(screen.getByRole("button", {name: "Add a reaction"}))
        await user.click(screen.getByRole("button", {name: "Clown"}))

        expect(backend.react).toHaveBeenCalledWith({messageId: "m1", reaction: Reaction.CLOWN, on: true})
        expect(await screen.findByRole("button", {name: "Clown: 1", pressed: true})).toBeDefined()
        expect(screen.queryByRole("group", {name: "Reactions"})).toBeNull()
    })

    it("takes its own reaction off from the count", async () => {
        const {backend} = stubBackend([{...message("m1", "gm"), reactions: [clown(2, true)]}])
        const {user} = setup(backend)

        await user.click(await screen.findByRole("button", {name: "Clown: 2", pressed: true}))

        expect(backend.react).toHaveBeenCalledWith({messageId: "m1", reaction: Reaction.CLOWN, on: false})
    })

    it("keeps its own mark through a count from the stream, which knows nobody", async () => {
        const {backend, broadcastReactions} = stubBackend([{...message("m1", "gm"), reactions: [clown(1, true)]}])
        setup(backend)
        await screen.findByRole("button", {name: "Clown: 1", pressed: true})

        act(() => broadcastReactions({messageId: "m1", reactions: [clown(3, false)], version: 2}))

        expect(await screen.findByRole("button", {name: "Clown: 3", pressed: true})).toBeDefined()
    })

    it("drops a frame older than the reactions it shows", async () => {
        const {backend, broadcastReactions} = stubBackend([{...message("m1", "gm"), reactions: [clown(4, false)], reactionsVersion: 5}])
        setup(backend)
        await screen.findByRole("button", {name: "Clown: 4"})

        act(() => broadcastReactions({messageId: "m1", reactions: [clown(1, false)], version: 3}))

        expect(screen.getByRole("button", {name: "Clown: 4"})).toBeDefined()
    })

    it("undoes a reaction the server refused", async () => {
        const {backend} = stubBackend([message("m1", "gm")])
        backend.react.mockRejectedValueOnce(new ChatRejectedError())
        const {user} = setup(backend)
        await screen.findByText("gm")

        await user.click(screen.getByRole("button", {name: "Add a reaction"}))
        await user.click(screen.getByRole("button", {name: "Clown"}))

        await waitFor(() => expect(screen.queryByRole("button", {name: /^Clown: /})).toBeNull())
    })

    it("undoes a reaction refused to a muted player, and says it is muted", async () => {
        const {backend} = stubBackend([message("m1", "gm")])
        backend.react.mockRejectedValueOnce(new ChatMutedError(Date.now() + 3600_000))
        vi.spyOn(console, "error").mockImplementation(() => {})
        const {user} = setup(backend)
        await screen.findByText("gm")

        await user.click(screen.getByRole("button", {name: "Add a reaction"}))
        await user.click(screen.getByRole("button", {name: "Clown"}))

        await waitFor(() => expect(screen.queryByRole("button", {name: /^Clown: /})).toBeNull())
        expect((await screen.findByRole("alert")).textContent).toMatch(/^You are muted until /)
    })

    it("closes the picker on Escape", async () => {
        const {backend} = stubBackend([message("m1", "gm")])
        const {user} = setup(backend)
        await screen.findByText("gm")

        await user.click(screen.getByRole("button", {name: "Add a reaction"}))
        await user.keyboard("{Escape}")

        expect(screen.queryByRole("group", {name: "Reactions"})).toBeNull()
    })
})

describe("ChatPanel replies", () => {
    beforeEach(() => window.localStorage.setItem(
        CHAT_IDENTITY_STORAGE_KEY,
        JSON.stringify({authorId: "author-1"}),
    ))

    const replying = () => document.querySelector<HTMLElement>(".chat-replying")

    const open = async () => {
        const stub = stubBackend([message("m1", "who holds the north"), message("m2", "gm", 1_700_000_001_000)])
        const view = setup(stub.backend)
        await screen.findByText("who holds the north")
        return {...stub, ...view}
    }

    it("sends a reply to the message whose button was pressed, and shows it above the box meanwhile", async () => {
        const {backend, user} = await open()

        await user.click(within(item("who holds the north")).getByRole("button", {name: "Reply to Ana"}))

        expect(replying()?.textContent).toBe("Replying to Ana who holds the north")
        expect(document.activeElement).toBe(messageBox())

        await user.type(messageBox(), "we do{Enter}")

        expect(backend.sendMessage).toHaveBeenCalledWith({
            authorId: "author-1",
            countryCode: "fr",
            text: "we do",
            replyTo: "m1",
        })
        await waitFor(() => expect(replying()).toBeNull())
    })

    it("takes the focus back to the box when the same message is picked again", async () => {
        const {user} = await open()
        const reply = within(item("gm")).getByRole("button", {name: "Reply to Ana"})

        await user.click(reply)
        messageBox().blur()
        await user.click(reply)

        expect(document.activeElement).toBe(messageBox())
        expect(replying()?.textContent).toBe("Replying to Ana gm")
    })

    it("sends a plain message again once the reply is sent", async () => {
        const {backend, user} = await open()
        await user.click(within(item("gm")).getByRole("button", {name: "Reply to Ana"}))
        await user.type(messageBox(), "gm gm{Enter}")
        await waitFor(() => expect(backend.sendMessage).toHaveBeenCalledTimes(1))

        await user.type(messageBox(), "anyone here{Enter}")

        await waitFor(() => expect(backend.sendMessage).toHaveBeenCalledTimes(2))
        expect(backend.sendMessage.mock.calls[1][0].replyTo).toBeUndefined()
    })

    it("answers the last message picked", async () => {
        const {backend, user} = await open()

        await user.click(within(item("who holds the north")).getByRole("button", {name: "Reply to Ana"}))
        await user.click(within(item("gm")).getByRole("button", {name: "Reply to Ana"}))
        await user.type(messageBox(), "gm{Enter}")

        expect(backend.sendMessage.mock.calls[0][0].replyTo).toBe("m2")
    })

    it("drops the reply from its cross, and from Escape", async () => {
        const {backend, user} = await open()

        await user.click(within(item("gm")).getByRole("button", {name: "Reply to Ana"}))
        await user.click(screen.getByRole("button", {name: "Cancel the reply"}))
        expect(replying()).toBeNull()

        await user.click(within(item("gm")).getByRole("button", {name: "Reply to Ana"}))
        await user.keyboard("{Escape}")
        expect(replying()).toBeNull()

        await user.type(messageBox(), "never mind{Enter}")
        expect(backend.sendMessage.mock.calls[0][0].replyTo).toBeUndefined()
    })

    it("keeps the reply and the text when the message is refused", async () => {
        const {backend, user} = await open()
        backend.sendMessage.mockRejectedValueOnce(new ChatRateLimitedError())

        await user.click(within(item("gm")).getByRole("button", {name: "Reply to Ana"}))
        await user.type(messageBox(), "gm gm{Enter}")

        await screen.findByRole("alert")
        expect(replying()?.textContent).toBe("Replying to Ana gm")
        expect(messageBox()).toHaveProperty("value", "gm gm")
    })

    it("drops a reply to a message that is gone, says so, and keeps the text", async () => {
        const {backend, user} = await open()
        backend.sendMessage.mockRejectedValueOnce(new ChatMessageGoneError())

        await user.click(within(item("gm")).getByRole("button", {name: "Reply to Ana"}))
        await user.type(messageBox(), "gm gm{Enter}")

        expect((await screen.findByRole("alert")).textContent).toBe("The message you replied to is gone.")
        expect(replying()).toBeNull()
        expect(messageBox()).toHaveProperty("value", "gm gm")
    })
})

describe("ChatPanel reports", () => {
    const theirs = message("m1", "you all suck")
    const mine = {...message("m2", "no u", 1_700_000_001_000), authorAccountId: OWN_ACCOUNT, authorName: "ana_1"}

    it("reports a message from its flag, with the details, by its own id, and marks it reported", async () => {
        const {backend} = stubBackend([theirs, mine])
        const reportPlayer = vi.fn(async () => undefined)
        render(<ChatPanel backend={backend} country={france} username="ana_1" accountId={OWN_ACCOUNT} reporter={{reportPlayer}}/>)
        await screen.findByText("you all suck")
        const user = userEvent.setup()

        expect(within(item("no u")).queryByRole("button", {name: /^Report/})).toBeNull()
        await user.click(within(item("you all suck")).getByRole("button", {name: "Report this message from Ana"}))
        const dialog = screen.getByRole("dialog", {name: "Report a message"})
        expect(within(dialog).getByText("you all suck")).toBeDefined()

        await user.type(within(dialog).getByRole("textbox", {name: "Details"}), "every day")
        await user.click(within(dialog).getByRole("button", {name: "Send"}))

        expect(reportPlayer.mock.calls).toEqual([[{accountId: "account-ana", reason: "chat-abuse", text: "every day", messageId: "m1"}]])
        expect((await within(dialog).findByRole("status")).textContent).toBe("Sent. Thank you!")
        await user.click(within(dialog).getByText("Close"))
        expect(screen.queryByRole("dialog")).toBeNull()
        expect(within(item("you all suck")).getByRole("img", {name: "Reported"})).toBeDefined()
    })

    it("closes the report on Escape, and keeps the phone's sheet open", async () => {
        const {backend} = stubBackend([theirs])
        const onOpenChange = vi.fn()
        render(<ChatPanel backend={backend} country={france} compact open onOpenChange={onOpenChange} reporter={{reportPlayer: vi.fn()}}/>)
        await screen.findByText("you all suck")
        const user = userEvent.setup()

        await user.click(within(item("you all suck")).getByRole("button", {name: "Report this message from Ana"}))
        await user.keyboard("{Escape}")

        expect(screen.queryByRole("dialog", {name: "Report a message"})).toBeNull()
        expect(onOpenChange).not.toHaveBeenCalled()
    })
})

describe("ChatPanel size", () => {
    const wanted = () => ({
        width: document.documentElement.style.getPropertyValue(WANTED_WIDTH),
        height: document.documentElement.style.getPropertyValue(WANTED_HEIGHT),
    })

    beforeEach(() => window.localStorage.setItem(CHAT_SIZE_STORAGE_KEY, JSON.stringify({width: 520, height: 640})))

    it("opens at the size the player left it", async () => {
        const {backend} = stubBackend()
        setup(backend)
        await screen.findByRole("textbox", {name: "Message"})

        expect(wanted()).toEqual({width: "520px", height: "640px"})
    })

    it("goes back to its first size on a double-click of an edge", async () => {
        const {backend} = stubBackend()
        const {container} = setup(backend)
        await screen.findByRole("textbox", {name: "Message"})

        fireEvent.doubleClick(container.querySelector(".chat-resize-corner")!)

        expect(wanted()).toEqual({width: "", height: ""})
        expect(window.localStorage.getItem(CHAT_SIZE_STORAGE_KEY)).toBeNull()
    })

    it("hands the page its size back once it is gone", async () => {
        const {backend} = stubBackend()
        const {unmount} = setup(backend)
        await screen.findByRole("textbox", {name: "Message"})

        unmount()

        expect(wanted()).toEqual({width: "", height: ""})
    })
})

describe("ChatPanel on a desktop", () => {
    it("folds from its header and unfolds again", async () => {
        const {backend} = stubBackend([message("a", "who took Brittany")])
        const {user} = setup(backend)
        expect(await screen.findByText("who took Brittany")).toBeDefined()

        await user.click(screen.getByRole("button", {name: "Fold the chat"}))
        expect(screen.queryByText("who took Brittany")).toBeNull()

        await user.click(screen.getByRole("button", {name: /Chat/}))
        expect(await screen.findByText("who took Brittany")).toBeDefined()
    })
})

describe("ChatPanel's players", () => {
    const players = [
        {key: "k1", accountId: "account-ana", name: "ana", countryCode: "fr", guest: false, admin: false, color: NameColor.UNSPECIFIED, streak: 0},
        {key: "k2", accountId: "account-b0b", name: "guest_b0b0b0", countryCode: "de", guest: true, admin: false, color: NameColor.UNSPECIFIED, streak: 0},
    ]

    it("shows who is online beside the chat, and goes back to it", async () => {
        const {backend} = stubBackend([message("a", "who took Brittany")])
        const user = userEvent.setup()
        render(<ChatPanel backend={backend} country={france} players={players}/>)
        await screen.findByText("who took Brittany")

        await user.click(screen.getByRole("tab", {name: "2 players online"}))
        expect(screen.getByText("ana")).toBeDefined()
        expect(screen.queryByText("who took Brittany")).toBeNull()

        await user.click(screen.getByRole("tab", {name: "Chat"}))
        expect(await screen.findByText("who took Brittany")).toBeDefined()
    })

    it("counts on the Chat tab what lands while the list is shown, until the chat is shown again", async () => {
        const T = 1_700_000_000_000
        const {backend, broadcast} = stubBackend([message("a", "who took Brittany", T)], [], T)
        const user = userEvent.setup()
        render(<ChatPanel backend={backend} country={france} players={players}/>)
        await screen.findByText("who took Brittany")
        await user.click(screen.getByRole("tab", {name: "2 players online"}))

        act(() => broadcast(message("b", "we did", T + 1_000)))
        act(() => broadcast(message("c", "and Normandy", T + 2_000)))

        expect(await screen.findByRole("tab", {name: "Chat, 2 new messages"})).toBeDefined()
        await user.click(screen.getByRole("tab", {name: /Chat/}))
        expect(await screen.findByText("and Normandy")).toBeDefined()
        expect(screen.getByRole("tab", {name: "Chat"})).toBeDefined()
    })

    it("counts one player in the singular", () => {
        render(<ChatPanel backend={stubBackend().backend} country={france} players={players.slice(0, 1)}/>)
        expect(screen.getByRole("tab", {name: "1 player online"})).toBeDefined()
    })

    it("says so when nobody is playing", async () => {
        const user = userEvent.setup()
        render(<ChatPanel backend={stubBackend().backend} country={france} players={[]}/>)

        await user.click(screen.getByRole("tab", {name: "0 players online"}))
        expect(screen.getByText("Nobody is playing right now.")).toBeDefined()
    })

    it("offers no list without a roster", () => {
        render(<ChatPanel backend={stubBackend().backend} country={france}/>)
        expect(screen.queryByRole("tab")).toBeNull()
    })
})

describe("ChatPanel's Discord bar", () => {
    const discord = () => screen.queryByRole("link", {name: "Join Discord"})

    it("sits over the chat, and not over who is online", async () => {
        const user = userEvent.setup()
        render(<ChatPanel backend={stubBackend().backend} country={france} players={[]}/>)
        expect(discord()).not.toBeNull()

        await user.click(screen.getByRole("tab", {name: "0 players online"}))
        expect(discord()).toBeNull()
    })

    it("is in the phone's sheet too", () => {
        render(<ChatPanel backend={stubBackend().backend} country={france} compact open/>)
        expect(discord()).not.toBeNull()
    })

    it("stays gone once its cross is pressed, through a fold of the chat", async () => {
        const {user} = setup(stubBackend().backend)

        await user.click(screen.getByRole("button", {name: "Hide the links"}))
        await user.click(screen.getByRole("button", {name: "Fold the chat"}))
        await user.click(screen.getByRole("button", {name: /Chat/}))

        expect(discord()).toBeNull()
    })
})

describe("ChatPanel on a phone", () => {
    const phone = (backend: ChatBackend, open: boolean, onOpenChange = vi.fn(), onUnread = vi.fn()) =>
        render(<ChatPanel backend={backend} country={france} compact open={open} onOpenChange={onOpenChange} onUnread={onUnread}/>)

    it("is a sheet while it is open, and closes on its ×", async () => {
        const {backend} = stubBackend([message("a", "who took Brittany")])
        const onOpenChange = vi.fn()
        phone(backend, true, onOpenChange)

        expect(await screen.findByText("who took Brittany")).toBeDefined()
        fireEvent.click(screen.getByRole("button", {name: "Close"}))
        expect(onOpenChange).toHaveBeenCalledWith(false)
    })

    it("shows nothing while it is closed and nobody spoke", async () => {
        const {backend} = stubBackend([message("a", "seen already")])
        const {container} = phone(backend, false)
        await waitFor(() => expect(backend.getHistory).toHaveBeenCalled())

        expect(container.innerHTML).toBe("")
    })

    it("counts what arrived while it was closed, for the tab", async () => {
        const {backend, broadcast} = stubBackend([message("a", "seen already")])
        const onUnread = vi.fn()
        phone(backend, false, vi.fn(), onUnread)
        await waitFor(() => expect(backend.getHistory).toHaveBeenCalled())

        broadcast(message("b", "and one more", 1_700_000_200_000))

        await waitFor(() => expect(onUnread).toHaveBeenLastCalledWith(1))
    })

    it("peeks at a new message for a moment, and opens on it", async () => {
        const {backend, broadcast} = stubBackend([message("a", "seen already")])
        const onOpenChange = vi.fn()
        phone(backend, false, onOpenChange)
        await waitFor(() => expect(backend.getHistory).toHaveBeenCalled())

        broadcast(message("b", "who took Brittany", 1_700_000_200_000))

        const peek = await screen.findByRole("button", {name: "Open the chat: Ana, who took Brittany"})
        fireEvent.click(peek)
        expect(onOpenChange).toHaveBeenCalledWith(true)
    })

    it("lets the peek go after a few seconds", async () => {
        vi.useFakeTimers({shouldAdvanceTime: true})
        const {backend, broadcast} = stubBackend([message("a", "seen already")])
        phone(backend, false)
        await waitFor(() => expect(backend.getHistory).toHaveBeenCalled())

        broadcast(message("b", "who took Brittany", 1_700_000_200_000))
        await screen.findByRole("button", {name: /Open the chat/})

        act(() => vi.advanceTimersByTime(4_000))
        expect(screen.queryByRole("button", {name: /Open the chat/})).toBeNull()
        vi.useRealTimers()
    })
})
