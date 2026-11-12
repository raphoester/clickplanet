// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {BombAnnouncement, ChatMessage, FlagMissionAnnouncement, KnockOutAnnouncement} from "../../backends/chat.ts"
import {quoteOf} from "../../domain/chatLog.ts"
import ChatLog from "./ChatLog.tsx"
import {QUOTE_MAX_LENGTH} from "./QuoteLine.tsx"
import {NameColor} from "../../backends/player.ts"
import {wordFilterOf} from "../../domain/wordFilter.ts"
import {WordFilterContext} from "../crazygames/wordFilter.ts"
import {usePlayerCard} from "../players/playerCardStore.ts"
import {useBadges} from "../badges/badgesStore.ts"
import {useChat} from "./chatStore.ts"
import {useNameplates} from "../nameplates/nameplatesStore.ts"

// What the nameplates say of an author: a guest by its prefix, no color and no crown unless told.
function named(name: string, plate: {color?: NameColor, admin?: boolean} = {}) {
    useNameplates.getState().learn([{
        accountId: `account-${name}`, name, color: plate.color ?? NameColor.UNSPECIFIED, admin: plate.admin ?? false,
        guest: name.startsWith("guest_"),
    }])
}

beforeEach(() => ["Ana", "Bo", "Cy", "kiran_07", "guest_Bo", "guest_a1b2c3"].forEach((name) => named(name)))

afterEach(() => {
    cleanup()
    usePlayerCard.setState({player: undefined})
    useChat.setState(useChat.getInitialState(), true)
    useBadges.getState().serve(undefined)
    useNameplates.getState().serve(undefined)
})

const message = (id: string, authorName: string, countryCode: string): ChatMessage =>
    ({id, sentAt: Date.UTC(2026, 8, 17, 12), authorAccountId: `account-${authorName}`, countryCode, text: "hello", reactions: [], reactionsVersion: 0})

describe("ChatLog's flag mission line", () => {
    const done: FlagMissionAnnouncement = {
        kind: "flag_mission", id: "a1", announcedAt: Date.UTC(2026, 10, 5, 12), country: "fr", points: 6,
        order: {kind: "takeIn", country: "br", goal: 3000, taken: 3000},
    }

    it("says which flag finished which mission and the points it won", () => {
        render(<ChatLog loading={false} messages={[]} announcements={[done]}/>)

        const line = screen.getByText((_, element) => element?.classList.contains("chat-announcement-text") ?? false)
        expect(line.textContent).toMatch(/^France completed its flag mission: Take 3,000 tiles in\s+Brazil \(\+6 season points\)$/)
    })

    it("says only that it was done for a kind this build does not know", () => {
        render(<ChatLog loading={false} messages={[]} announcements={[{...done, order: undefined, points: 1}]}/>)

        const line = screen.getByText((_, element) => element?.classList.contains("chat-announcement-text") ?? false)
        expect(line.textContent).toBe("France completed its flag mission (+1 season point)")
    })
})

describe("ChatLog's knock-out line", () => {
    it("says which flag knocked out which", () => {
        const knockOut: KnockOutAnnouncement = {kind: "knock_out", id: "a1", announcedAt: Date.UTC(2026, 10, 11, 20), country: "fr", by: "de"}
        render(<ChatLog loading={false} messages={[]} announcements={[knockOut]}/>)

        const line = screen.getByText((_, element) => element?.classList.contains("chat-announcement-text") ?? false)
        expect(line.textContent).toBe("Germany knocked out France")
    })
})

describe("ChatLog's bomb line", () => {
    const bomb: BombAnnouncement = {kind: "bomb", id: "a1", announcedAt: Date.UTC(2026, 10, 11, 20), country: "fr", tile: 42, cleared: 3}
    const lineOf = (announcement: BombAnnouncement) => {
        render(<ChatLog loading={false} messages={[]} announcements={[announcement]}/>)
        return screen.getByText((_, element) => element?.classList.contains("chat-announcement-text") ?? false).textContent
    }

    it("counts the tiles a bomb cleared where no land is named", () => {
        expect(lineOf(bomb)).toBe("France bombed 3 tiles")
    })

    it("says one tile alone", () => {
        expect(lineOf({...bomb, cleared: 1})).toBe("France bombed 1 tile")
    })

    it("says a bomb that cleared nothing fell on empty land", () => {
        expect(lineOf({...bomb, cleared: 0})).toBe("France bombed empty land")
    })
})

describe("ChatLog", () => {
    const dancing = {url: "https://static.klipy.com/ii/a/sm.webp", width: 220, height: 124}

    it("draws a GIF at its size, no wider than the line, and a reply to it says GIF", () => {
        const gif = {...message("1", "Ana", "fr"), text: "", gif: dancing}
        const reply = {...message("2", "Bo", "de"), text: "nice", replyTo: quoteOf(gif)}
        render(<ChatLog loading={false} messages={[gif, reply]}/>)

        const image = screen.getByRole("img", {name: "GIF"})
        expect(image.getAttribute("src")).toBe(dancing.url)
        expect([image.getAttribute("width"), image.getAttribute("height")]).toEqual(["200", "113"])
        expect(screen.getByText("GIF", {selector: ".chat-quote-text"})).toBeDefined()
    })

    it("shows a removed GIF as removed, with no image", () => {
        render(<ChatLog loading={false} messages={[{...message("1", "Ana", "fr"), text: "", removed: true}]}/>)

        expect(screen.queryByRole("img", {name: "GIF"})).toBeNull()
        expect(screen.getByText("This message has been deleted")).toBeDefined()
    })

    it("opens the author of a message from its name, and tells a guest by its nameplate", async () => {
        render(<ChatLog loading={false}
                        messages={[message("1", "Ana", "fr"), message("2", "guest_Bo", "de")]}/>)

        await userEvent.click(screen.getByRole("button", {name: "Ana"}))
        expect(usePlayerCard.getState().player).toEqual(
            {accountId: "account-Ana", name: "Ana", countryCode: "fr", guest: false, admin: false, color: NameColor.UNSPECIFIED})
        await userEvent.click(screen.getByRole("button", {name: "guest_Bo"}))
        expect(usePlayerCard.getState().player).toEqual(
            {accountId: "account-guest_Bo", name: "guest_Bo", countryCode: "de", guest: true, admin: false, color: NameColor.UNSPECIFIED})
    })

    it("shows a guest's name once, and nothing of its address", () => {
        const {container} = render(<ChatLog loading={false} messages={[message("1", "guest_a1b2c3", "de")]}/>)

        expect(container.querySelector(".chat-message-head")?.textContent).not.toContain("#")
        expect(screen.getAllByText(/a1b2c3/)).toHaveLength(1)
    })

    it("crowns an admin's message, and keeps it on the player it opens", async () => {
        named("Ana", {admin: true})
        render(<ChatLog loading={false} messages={[message("1", "Ana", "fr"), message("2", "kiran_07", "in")]}/>)

        expect(screen.getAllByRole("img", {name: "Admin"})).toHaveLength(1)
        await userEvent.click(screen.getByRole("button", {name: "Ana"}))
        expect(usePlayerCard.getState().player).toEqual({
            accountId: "account-Ana", name: "Ana", countryCode: "fr", guest: false, admin: true, color: NameColor.UNSPECIFIED,
        })
    })

    it("lights a flame beside a streak of three days or more, from the badge kept for the author's account", () => {
        useBadges.setState({badges: {"account-Ana": {streak: 12}, "account-kiran_07": {streak: 2}}})
        render(<ChatLog loading={false} messages={[message("1", "Ana", "fr"), message("2", "kiran_07", "in")]}/>)

        expect(screen.getAllByRole("img", {name: /streak/})).toHaveLength(1)
        expect(screen.getByRole("img", {name: "12-day streak"}).textContent).toBe("12")
    })

    it("shows the title its author wears beside the name, from the badge kept for its account", () => {
        const warlord = {id: "warlord", name: "Warlord", rank: {trackId: "conquest", trackName: "Conquest", number: 3, count: 5}}
        useBadges.setState({badges: {"account-Ana": {streak: 0, wornTitle: warlord}}})
        render(<ChatLog loading={false} messages={[message("1", "Ana", "fr"), message("2", "kiran_07", "in")]}/>)

        expect(screen.getAllByRole("img", {name: "Warlord"})).toHaveLength(1)
        expect(screen.getByRole("img", {name: "Warlord"}).querySelector(".title-emblem")).not.toBeNull()
    })

    it("shows no badge beside a message with no account: kept before accounts", () => {
        useBadges.setState({badges: {"": {streak: 12, wornTitle: {id: "og", name: "OG"}}}})
        render(<ChatLog loading={false} messages={[{...message("1", "Ana", "fr"), authorAccountId: ""}]}/>)

        expect(screen.queryByRole("img", {name: /streak/})).toBeNull()
        expect(screen.queryByRole("img", {name: "OG"})).toBeNull()
        expect(screen.getByText("[deleted]")).toBeDefined()
    })

    it("names an account nobody can name [deleted], in grey, with nothing to open or report", () => {
        useNameplates.setState({nameplates: {...useNameplates.getState().nameplates, "account-gone": null}})
        const {container} = render(<ChatLog loading={false} onReport={vi.fn()} messages={[message("1", "gone", "fr")]}/>)

        expect(screen.getByText("[deleted]")).toBeDefined()
        expect(screen.queryByRole("button", {name: "[deleted]"})).toBeNull()
        expect(screen.queryAllByRole("button", {name: /^Report this message/})).toHaveLength(0)
        expect(container.querySelector<HTMLElement>(".chat-message")!.style.getPropertyValue("--author-chroma")).toBe("0")
        expect(screen.getByText("hello")).toBeDefined()
    })

    it("paints a name in the color its player chose, and grey with none or as a guest", () => {
        named("Ana", {color: NameColor.TEAL})
        named("guest_Bo", {color: NameColor.TEAL})
        const chosen = message("1", "Ana", "fr")
        const guest = message("2", "guest_Bo", "de")
        const none = message("3", "Cy", "it")
        const {container} = render(<ChatLog loading={false} messages={[chosen, guest, none]}/>)

        const [ana, bo, cy] = [...container.querySelectorAll<HTMLElement>(".chat-message")]
        expect(ana.style.getPropertyValue("--author-hue")).toBe("165")
        expect(ana.style.getPropertyValue("--author-chroma")).toBe("")
        expect(bo.style.getPropertyValue("--author-chroma")).toBe("0")
        expect(cy.style.getPropertyValue("--author-chroma")).toBe("0")
    })

    it("links an address a player wrote, in a new tab, and leaves a guest's as text", () => {
        const linked = {...message("1", "Ana", "fr"), text: "see https://clickplanet.lol/play now"}
        const plain = {...message("2", "guest_a1b2c3", "de"), text: "see https://other.example"}
        render(<ChatLog loading={false} messages={[linked, plain]}/>)

        const links = screen.getAllByRole("link")
        expect(links).toHaveLength(1)
        expect(links[0].textContent).toBe("https://clickplanet.lol/play")
        expect(links[0].getAttribute("href")).toBe("https://clickplanet.lol/play")
        expect(links[0].getAttribute("target")).toBe("_blank")
        expect(links[0].getAttribute("rel")).toBe("noopener noreferrer nofollow ugc")
        expect(links[0].closest(".chat-message-text")!.textContent).toBe("see https://clickplanet.lol/play now")
        expect(screen.getByText("see https://other.example")).toBeDefined()
    })

    describe("a report", () => {
        const first = {...message("1", "Ana", "fr"), text: "hi"}
        const second = {...message("2", "Ana", "fr"), text: "you all suck"}

        it("hands up the message it was pressed on, the second of its author's run too", async () => {
            const onReport = vi.fn()
            const {container} = render(<ChatLog loading={false} onReport={onReport} messages={[first, second]}/>)
            const [, last] = [...container.querySelectorAll<HTMLElement>(".chat-message")]
            expect(last.querySelector(".chat-message-head")).toBeNull()

            await userEvent.click(within(last).getByRole("button", {name: "Report this message from Ana"}))

            expect(onReport.mock.calls).toEqual([[second]])
        })

        it("shows a message already reported as reported, with nothing to press", () => {
            useChat.setState({reported: new Set(["2"])})
            render(<ChatLog loading={false} onReport={vi.fn()} messages={[first, second]}/>)

            expect(screen.getAllByRole("button", {name: "Report this message from Ana"})).toHaveLength(1)
            expect(screen.getByRole("img", {name: "Reported"}).closest("[data-message]")?.getAttribute("data-message")).toBe("2")
        })

        it("is not offered on the player's own message, nor on one with no account", () => {
            const gone = {...message("3", "Cy", "it"), authorAccountId: ""}
            const theirs = message("4", "Bo", "de")
            render(<ChatLog loading={false}
                            onReport={vi.fn()}
                            own={(shown) => shown.authorAccountId === "account-Ana"}
                            messages={[first, gone, theirs]}/>)

            expect(screen.getAllByRole("button", {name: /^Report this message/}).map(button => button.getAttribute("aria-label")))
                .toEqual(["Report this message from Bo"])
        })

        it("is not offered with nobody to send it to", () => {
            render(<ChatLog loading={false} messages={[first, second]}/>)

            expect(screen.queryAllByRole("button", {name: /^Report this message/})).toHaveLength(0)
        })
    })

    describe("a reply", () => {
        const question = {...message("1", "Ana", "fr"), text: "who holds the north"}
        const answer = {...message("2", "Bo", "de"), text: "we do", replyTo: quoteOf(question)}

        beforeEach(() => named("Ana", {color: NameColor.TEAL}))

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

    it("masks the words of the filter in the text, the names and the quotes", () => {
        named("ass")
        const said = {...message("1", "ass", "fr"), text: "fuck that ass"}
        const reply = {...message("2", "Bo", "de"), text: "no", replyTo: quoteOf(said)}
        const mute = {kind: "mute" as const, id: "m", announcedAt: Date.UTC(2026, 8, 17, 12), name: "fuck", muteKind: "chat" as const, seconds: 3600}
        const {container} = render(<WordFilterContext.Provider value={wordFilterOf(["ass", "fuck"])}>
            <ChatLog loading={false} onReply={vi.fn()} messages={[said, reply]} announcements={[mute]}/>
        </WordFilterContext.Provider>)

        expect(container.textContent).not.toMatch(/fuck|ass/)
        expect(container.querySelector(".chat-message-author")?.textContent).toBe("a*s")
        expect(screen.getByRole("button", {name: "Reply to a*s"})).toBeDefined()
        expect(container.querySelector(".chat-quote-text")?.textContent).toBe("f**k that a*s")
        expect(screen.getByText("f**k", {selector: "strong"})).toBeDefined()
    })
})
