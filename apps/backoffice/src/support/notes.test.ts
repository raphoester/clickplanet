import {Timestamp} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {AT} from "../fakeAdmin.ts"
import {Note, NoteAnswer, NoteReply} from "../gen/grpc/support/v1/admin_pb.ts"
import {conversationOf, lastWordsOf, latestFirst, statusOf} from "./notes.ts"

const minutes = (n: number) => Timestamp.fromDate(new Date(AT.getTime() + n * 60_000))

// Answered, closed, then replied to.
const note = new Note({
    id: "note-1", text: "more flags", submittedAt: minutes(0), replyAddress: "ada@example.com",
    answers: [new NoteAnswer({text: "Which ones?", sentAt: minutes(10)})],
    replies: [new NoteReply({text: "Tibet", from: "ada@example.com", receivedAt: minutes(30)})],
    closedAt: minutes(20),
})

describe("a note's conversation", () => {
    it("is the note, then the answers, the replies and the close, by time", () => {
        expect(conversationOf(note).map(entry => entry.kind)).toEqual(["note", "answer", "closed", "reply"])
    })

    it("is the note alone when nobody wrote about it", () => {
        expect(conversationOf(new Note({text: "hi"}))).toEqual([{kind: "note", at: undefined, text: "hi"}])
    })

    it("ends with what the player wrote last", () => {
        expect(lastWordsOf(note)).toBe("Tibet")
        expect(lastWordsOf(new Note({text: "hi"}))).toBe("hi")
    })
})

describe("where a note stands", () => {
    it("is new or replied while it waits", () => {
        expect(statusOf(note, true)).toBe("Replied")
        expect(statusOf(new Note({text: "hi", replyAddress: "ada@example.com"}), true)).toBe("New")
    })

    it("is answered or closed, whichever came last, once it waits no more", () => {
        const answered = new Note({...note, replies: [], closedAt: undefined})
        expect(statusOf(answered, false)).toBe("Answered")
        expect(statusOf(new Note({...answered, closedAt: minutes(15)}), false)).toBe("Closed")
    })

    it("has no address when it cannot be answered", () => {
        expect(statusOf(new Note({text: "hi"}), false)).toBe("No address")
    })
})

describe("the notes in the inbox", () => {
    it("come the last written to first", () => {
        const old = new Note({id: "old", submittedAt: minutes(-100), replies: [new NoteReply({receivedAt: minutes(40)})]})
        const fresh = new Note({id: "fresh", submittedAt: minutes(5)})
        expect(latestFirst([fresh, note, old]).map(each => each.id)).toEqual(["old", "note-1", "fresh"])
    })
})
