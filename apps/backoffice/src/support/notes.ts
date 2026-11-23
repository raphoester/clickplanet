import {Timestamp} from "@bufbuild/protobuf"
import {Note} from "../gen/grpc/support/v1/admin_pb.ts"

export type Entry =
    | {kind: "note", at: Timestamp | undefined, text: string}
    | {kind: "answer", at: Timestamp | undefined, text: string}
    | {kind: "reply", at: Timestamp | undefined, text: string, from: string}
    | {kind: "closed", at: Timestamp | undefined}

// The note first, then the answers, the player's replies and the last close, by time.
export function conversationOf(note: Note): Entry[] {
    const after: Entry[] = [
        ...note.answers.map(answer => ({kind: "answer" as const, at: answer.sentAt, text: answer.text})),
        ...note.replies.map(reply => ({kind: "reply" as const, at: reply.receivedAt, text: reply.text, from: reply.from})),
        ...(note.closedAt === undefined ? [] : [{kind: "closed" as const, at: note.closedAt}]),
    ]
    return [{kind: "note", at: note.submittedAt, text: note.text}, ...after.sort((a, b) => msOf(a.at) - msOf(b.at))]
}

export function lastAtOf(note: Note): Timestamp | undefined {
    return conversationOf(note).at(-1)?.at
}

// What the player wrote last: its last reply, or else the note.
export function lastWordsOf(note: Note): string {
    return note.replies.at(-1)?.text ?? note.text
}

// Where a note stands, in a word. `open` is the server's: the note waits for an answer.
export function statusOf(note: Note, open: boolean): string {
    const last = conversationOf(note).at(-1)?.kind
    if (!open && note.replyAddress === "") return "No address"
    if (!open && last === "answer") return "Answered"
    if (!open && last === "closed") return "Closed"
    return last === "reply" ? "Replied" : "New"
}

export function latestFirst(notes: Note[]): Note[] {
    return [...notes].sort((a, b) => msOf(lastAtOf(b)) - msOf(lastAtOf(a)))
}

function msOf(at: Timestamp | undefined): number {
    return at?.toDate().getTime() ?? 0
}
