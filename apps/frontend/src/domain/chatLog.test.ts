import {describe, expect, it} from "vitest"
import {addMessages, CHAT_LOG_LIMIT, GROUP_WINDOW_MS, idsSince, startsGroup, unreadSince} from "./chatLog.ts"
import type {ChatMessage} from "../backends/chat.ts"

const message = (id: string, sentAt: number): ChatMessage => ({
    id,
    sentAt,
    authorName: "Ana",
    authorTag: "4f2ca1",
    countryCode: "fr",
    text: `message ${id}`,
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

describe("startsGroup", () => {
    const from = (author: string, tag: string, sentAt: number): ChatMessage =>
        ({...message("x", sentAt), authorName: author, authorTag: tag})

    it("opens a group on the first message there is", () => {
        expect(startsGroup(undefined, from("Ana", "4f2ca1", 0))).toBe(true)
    })

    it("keeps one author's run together", () => {
        const first = from("Ana", "4f2ca1", 0)
        const next = from("Ana", "4f2ca1", 1_000)

        expect(startsGroup(first, next)).toBe(false)
    })

    it("opens a group when somebody else speaks", () => {
        const ana = from("Ana", "4f2ca1", 0)
        const bo = from("Bo", "c0ffee", 1_000)

        expect(startsGroup(ana, bo)).toBe(true)
    })

    it("opens a group for a namesake with another tag", () => {
        const ana = from("Ana", "4f2ca1", 0)
        const otherAna = from("Ana", "c0ffee", 1_000)

        expect(startsGroup(ana, otherAna)).toBe(true)
    })

    it("opens a group again after a long enough silence", () => {
        const first = from("Ana", "4f2ca1", 0)

        expect(startsGroup(first, from("Ana", "4f2ca1", GROUP_WINDOW_MS))).toBe(false)
        expect(startsGroup(first, from("Ana", "4f2ca1", GROUP_WINDOW_MS + 1))).toBe(true)
    })
})
