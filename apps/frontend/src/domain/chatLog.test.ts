import {describe, expect, it} from "vitest"
import {
    accountSentUnder,
    addAnnouncements,
    addMessages,
    CHAT_LOG_LIMIT,
    ChatLogEntry,
    GROUP_WINDOW_MS,
    idsSince,
    interleave,
    nameSentUnder,
    newestAt,
    quoteOf,
    startsGroup,
    unreadSince,
    unseenAfter,
} from "./chatLog.ts"
import type {ChatAnnouncement, ChatMessage} from "../backends/chat.ts"
import {NameColor} from "../backends/player.ts"

const message = (id: string, sentAt: number): ChatMessage => ({
    id,
    sentAt,
    authorAccountId: "account-ana",
    authorName: "Ana",
    authorAdmin: false,
    authorColor: NameColor.UNSPECIFIED,
    authorStreak: 0,
    countryCode: "fr",
    text: `message ${id}`,
    reactions: [],
    reactionsVersion: 0,
})

const ids = (messages: readonly ChatMessage[]) => messages.map(m => m.id)

describe("addMessages", () => {
    it("appends what it has not seen", () => {
        const log = addMessages([], [message("a", 1)])

        expect(ids(addMessages(log, [message("b", 2)]))).toEqual(["a", "b"])
    })

    it("ignores a message it already holds, whatever it arrived through", () => {
        const log = addMessages([], [message("a", 1), message("b", 2)])

        expect(ids(addMessages(log, [message("a", 1)]))).toEqual(["a", "b"])
    })

    it("returns the same array when nothing was added, so React sees no change", () => {
        const log = addMessages([], [message("a", 1)])

        expect(addMessages(log, [message("a", 1)])).toBe(log)
        expect(addMessages(log, [])).toBe(log)
    })

    it("drops duplicates inside one batch", () => {
        expect(ids(addMessages([], [message("a", 1), message("a", 1)]))).toEqual(["a"])
    })

    it("orders on the time the server stamped, not on arrival", () => {
        const log = addMessages([], [message("late", 200)])

        expect(ids(addMessages(log, [message("early", 100)]))).toEqual(["early", "late"])
    })

    it("breaks a tie on the id, so equal timestamps stop swapping", () => {
        expect(ids(addMessages([], [message("b", 1), message("a", 1)]))).toEqual(["a", "b"])
    })

    it("keeps the newest when it runs past the limit", () => {
        const log = addMessages([], [message("a", 1), message("b", 2), message("c", 3)], 2)

        expect(ids(log)).toEqual(["b", "c"])
    })

    it("caps at 200 by default", () => {
        const many = Array.from({length: CHAT_LOG_LIMIT + 10}, (_, i) => message(`m${i}`, i))

        expect(addMessages([], many)).toHaveLength(CHAT_LOG_LIMIT)
    })
})

describe("unreadSince", () => {
    const log = [message("a", 1), message("b", 2), message("c", 3)]

    it("counts what arrived after the last seen message", () => {
        expect(unreadSince(log, "a")).toBe(2)
        expect(unreadSince(log, "c")).toBe(0)
    })

    it("counts the whole log when nothing has been seen", () => {
        expect(unreadSince(log, undefined)).toBe(3)
    })

    it("counts the whole log when the last seen message fell off the cap", () => {
        expect(unreadSince(log, "gone")).toBe(3)
    })

    it("is zero on an empty log", () => {
        expect(unreadSince([], undefined)).toBe(0)
    })
})

describe("idsSince", () => {
    const log = [message("a", 1), message("b", 2), message("c", 3)]

    it("names what arrived after the last seen message", () => {
        expect(idsSince(log, "a")).toEqual(["b", "c"])
    })

    it("names nothing when the last seen message is the last one", () => {
        expect(idsSince(log, "c")).toEqual([])
    })

    it("names the whole log when nothing has been seen", () => {
        expect(idsSince(log, undefined)).toEqual(["a", "b", "c"])
    })

    it("names as many ids as unreadSince counts", () => {
        for (const seen of [undefined, "a", "b", "c", "gone"]) {
            expect(idsSince(log, seen)).toHaveLength(unreadSince(log, seen))
        }
    })
})

describe("newestAt", () => {
    it("is the time of the newest line, message or announcement", () => {
        expect(newestAt([message("a", 1), message("b", 5)], [bomb("x", 3)])).toBe(5)
        expect(newestAt([message("a", 1)], [bomb("x", 3), bomb("y", 7)])).toBe(7)
    })

    it("is nothing for an empty log", () => {
        expect(newestAt([], [])).toBeUndefined()
    })
})

describe("unseenAfter", () => {
    const nobodys = () => false

    it("counts the messages and the announcements after the mark", () => {
        const log = [message("a", 1), message("b", 2), message("c", 3)]

        expect(unseenAfter(log, [bomb("x", 1), bomb("y", 4)], 1, nobodys)).toBe(3)
    })

    it("does not count a line at the mark itself", () => {
        expect(unseenAfter([message("a", 2)], [bomb("x", 2)], 2, nobodys)).toBe(0)
    })

    it("does not count the player's own messages", () => {
        const log = [message("a", 2), message("mine", 3)]

        expect(unseenAfter(log, [], 1, m => m.id === "mine")).toBe(1)
    })
})

describe("nameSentUnder", () => {
    const log = [
        {...message("a", 0), authorName: "guest_91aa3d"},
        {...message("b", 1_000), authorName: "guest_c0ffee"},
        {...message("c", 2_000), authorName: "ana_1"},
    ]

    it("answers the name the server gave the latest message this client sent", () => {
        expect(nameSentUnder(log, new Set(["b"]))).toBe("guest_c0ffee")
        expect(nameSentUnder(log, new Set(["b", "c"]))).toBe("ana_1")
    })

    it("knows nothing before this client sent anything still in the log", () => {
        expect(nameSentUnder(log, new Set())).toBeUndefined()
        expect(nameSentUnder(log, new Set(["gone"]))).toBeUndefined()
    })
})

describe("accountSentUnder", () => {
    const log = [
        {...message("a", 1), authorAccountId: "account-bo"},
        {...message("b", 2), authorAccountId: "account-guest"},
        {...message("c", 3), authorAccountId: "account-own"},
        {...message("d", 4), authorAccountId: ""},
    ]

    it("is the account of the latest message this client sent", () => {
        expect(accountSentUnder(log, new Set(["b"]))).toBe("account-guest")
        expect(accountSentUnder(log, new Set(["b", "c"]))).toBe("account-own")
    })

    it("skips a sent message that carries no account", () => {
        expect(accountSentUnder(log, new Set(["c", "d"]))).toBe("account-own")
        expect(accountSentUnder(log, new Set(["d"]))).toBeUndefined()
    })

    it("is unknown until a message was sent, and once it left the log", () => {
        expect(accountSentUnder(log, new Set())).toBeUndefined()
        expect(accountSentUnder(log, new Set(["gone"]))).toBeUndefined()
    })
})

describe("startsGroup", () => {
    const from = (author: string, sentAt: number): ChatMessage =>
        ({...message("x", sentAt), authorAccountId: `account-${author}`, authorName: author})

    it("opens a group on the first message there is", () => {
        expect(startsGroup(undefined, from("Ana", 0))).toBe(true)
    })

    it("keeps one author's run together", () => {
        const first = from("Ana", 0)
        const next = from("Ana", 1_000)

        expect(startsGroup(first, next)).toBe(false)
    })

    it("opens a group when somebody else speaks", () => {
        const ana = from("Ana", 0)
        const bo = from("Bo", 1_000)

        expect(startsGroup(ana, bo)).toBe(true)
    })

    it("opens a group when another account speaks under the same name", () => {
        const ana = from("Ana", 0)
        const other = {...from("Ana", 1_000), authorAccountId: "account-other"}

        expect(startsGroup(ana, other)).toBe(true)
        expect(startsGroup(ana, {...from("Ana", 1_000), authorAccountId: ""})).toBe(true)
    })

    it("opens a group when the same account speaks under a new name", () => {
        const ana = from("Ana", 0)
        const renamed = {...from("Ana", 1_000), authorName: "Ana_Lovelace"}

        expect(startsGroup(ana, renamed)).toBe(true)
    })

    it("keeps together the messages of one name that carry no account", () => {
        const first = {...from("[deleted]", 0), authorAccountId: ""}
        const next = {...from("[deleted]", 1_000), authorAccountId: ""}

        expect(startsGroup(first, next)).toBe(false)
    })

    it("opens a group again after a long enough silence", () => {
        const first = from("Ana", 0)

        expect(startsGroup(first, from("Ana", GROUP_WINDOW_MS))).toBe(false)
        expect(startsGroup(first, from("Ana", GROUP_WINDOW_MS + 1))).toBe(true)
    })

    it("opens a group on a reply, and lets the same author go on under it", () => {
        const first = from("Ana", 0)
        const reply = {...from("Ana", 1_000), replyTo: quoteOf(from("Bo", 0))}

        expect(startsGroup(first, reply)).toBe(true)
        expect(startsGroup(reply, from("Ana", 2_000))).toBe(false)
    })
})

describe("quoteOf", () => {
    it("is who said a message and all of what it said", () => {
        const said = {...message("m1", 0), authorName: "Bo", authorColor: NameColor.TEAL, text: "who holds the north"}

        expect(quoteOf(said)).toEqual({
            messageId: "m1",
            authorName: "Bo",
            authorColor: NameColor.TEAL,
            text: "who holds the north",
        })
    })
})

const bomb = (id: string, announcedAt: number): ChatAnnouncement =>
    ({kind: "bomb", id, announcedAt, country: "fr", cleared: 0})

const entryIds = (entries: ChatLogEntry[]) =>
    entries.map(entry => entry.kind === "message" ? entry.message.id : entry.announcement.id)

describe("addAnnouncements", () => {
    it("keeps each one once, oldest first", () => {
        const log = addAnnouncements([], [bomb("b", 2)])

        expect(addAnnouncements(log, [bomb("a", 1), bomb("b", 2)]).map(a => a.id)).toEqual(["a", "b"])
    })

    it("keeps only the newest when it is full", () => {
        expect(addAnnouncements([], [bomb("a", 1), bomb("b", 2), bomb("c", 3)], 2).map(a => a.id)).toEqual(["b", "c"])
    })
})

describe("interleave", () => {
    it("puts announcements between the messages by time", () => {
        expect(entryIds(interleave(
            [message("m1", 10), message("m2", 30)],
            [bomb("b0", 5), bomb("b1", 20), bomb("b2", 40)],
        ))).toEqual(["b0", "m1", "b1", "m2", "b2"])
    })

    it("puts the message first when both happened at once", () => {
        expect(entryIds(interleave([message("m", 10)], [bomb("b", 10)]))).toEqual(["m", "b"])
    })

    it("shows announcements alone when nobody said anything", () => {
        expect(entryIds(interleave([], [bomb("b", 1)]))).toEqual(["b"])
    })

    it("leaves out announcements older than a full log's oldest message", () => {
        expect(entryIds(interleave(
            [message("m1", 10), message("m2", 30)],
            [bomb("b0", 5), bomb("b1", 20)],
            2,
        ))).toEqual(["m1", "b1", "m2"])
    })

    it("keeps older announcements while the log has room", () => {
        expect(entryIds(interleave(
            [message("m1", 10), message("m2", 30)],
            [bomb("b0", 5)],
            3,
        ))).toEqual(["b0", "m1", "m2"])
    })
})
