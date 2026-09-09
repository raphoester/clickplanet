import {describe, expect, it} from "vitest"
import {addMessages, CHAT_LOG_LIMIT, unreadSince} from "./chatLog.ts"
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
