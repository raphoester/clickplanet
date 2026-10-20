import {describe, expect, it} from "vitest"
import {KeptMessage} from "../gen/grpc/chat/v1/admin_pb.ts"
import {AROUND, moreConversations, Read, Window} from "./theirConversations.ts"

// A chat of `length` messages, "m1" the oldest; the ones in `theirs` are Ana's, the others Bob's.
function chatOf(length: number, theirs: number[]): KeptMessage[] {
    return Array.from({length}, (_, i) => new KeptMessage({
        messageId: `m${i + 1}`,
        accountId: theirs.includes(i + 1) ? "ana" : "bob",
    }))
}

// Reads the chat as the server does, and counts the reads.
function reader(chat: KeptMessage[]): {read: Read, asked: string[]} {
    const asked: string[] = []
    const read: Read = (ask, limit) => {
        const cursor = ask.beforeMessageId ?? ask.afterMessageId ?? ""
        asked.push(ask.beforeMessageId !== undefined ? `before ${cursor}` : `after ${cursor}`)
        const at = chat.findIndex(message => message.messageId === cursor)
        const messages = ask.beforeMessageId !== undefined
            ? chat.slice(Math.max(0, at - limit), at)
            : chat.slice(at + 1, at + 1 + limit)
        return Promise.resolve({messages, more: messages.length === limit})
    }
    return {read, asked}
}

function theirsOf(chat: KeptMessage[]): KeptMessage[] {
    return chat.filter(message => message.accountId === "ana").reverse()
}

const ids = (window: Window) => window.lines.map(line => line.messageId)

describe("moreConversations", () => {
    it("shows one of their messages among the messages each side of it", async () => {
        const chat = chatOf(30, [15])

        const {shown, left} = await moreConversations([], theirsOf(chat), reader(chat).read, 3)

        expect(shown).toHaveLength(1)
        expect(ids(shown[0])).toEqual(Array.from({length: 2 * AROUND + 1}, (_, i) => `m${10 + i}`))
        expect(shown[0].moreBefore).toBe(true)
        expect(shown[0].moreAfter).toBe(true)
        expect(left).toEqual([])
    })

    it("needs no conversation of its own for a message another one already holds", async () => {
        const chat = chatOf(30, [12, 15])
        const {read, asked} = reader(chat)

        const {shown} = await moreConversations([], theirsOf(chat), read, 3)

        expect(shown).toHaveLength(1)
        expect(asked).toEqual(["before m15", "after m15"])
    })

    it("joins a conversation that reaches the one after it, so an exchange is one conversation", async () => {
        const chat = chatOf(40, [12, 20])

        const {shown} = await moreConversations([], theirsOf(chat), reader(chat).read, 3)

        expect(shown).toHaveLength(1)
        expect(ids(shown[0])).toEqual(Array.from({length: 19}, (_, i) => `m${7 + i}`))
        expect(shown[0].moreBefore).toBe(true)
        expect(shown[0].moreAfter).toBe(true)
    })

    it("keeps conversations a gap lies between apart, newest first", async () => {
        const chat = chatOf(60, [8, 20, 40])

        const {shown} = await moreConversations([], theirsOf(chat), reader(chat).read, 3)

        expect(shown.map(window => window.lines.find(line => line.accountId === "ana")?.messageId))
            .toEqual(["m40", "m20", "m8"])
    })

    it("builds as many conversations as asked, and leaves the rest of their messages for later", async () => {
        const chat = chatOf(100, [10, 30, 50, 70, 90])

        const first = await moreConversations([], theirsOf(chat), reader(chat).read, 2)
        expect(first.shown).toHaveLength(2)
        expect(first.left.map(message => message.messageId)).toEqual(["m50", "m30", "m10"])

        const next = await moreConversations(first.shown, first.left, reader(chat).read, 2)
        expect(next.shown).toHaveLength(4)
        expect(next.left.map(message => message.messageId)).toEqual(["m10"])
    })

    it("joins a new conversation to the last one shown, across two builds", async () => {
        const chat = chatOf(40, [12, 20])
        const theirs = theirsOf(chat)
        const first = await moreConversations([], theirs.slice(0, 1), reader(chat).read, 1)

        const next = await moreConversations(first.shown, theirs.slice(1), reader(chat).read, 1)

        expect(next.shown).toHaveLength(1)
        expect(ids(next.shown[0])[0]).toBe("m7")
    })

    it("tells the start and the end of the chat", async () => {
        const chat = chatOf(4, [2])

        const {shown} = await moreConversations([], theirsOf(chat), reader(chat).read, 1)

        expect(ids(shown[0])).toEqual(["m1", "m2", "m3", "m4"])
        expect(shown[0].moreBefore).toBe(false)
        expect(shown[0].moreAfter).toBe(false)
    })
})
