import {Timestamp} from "@bufbuild/protobuf"
import {Code, ConnectError} from "@connectrpc/connect"
import {SupportAdmin} from "../clients.ts"
import {Letter} from "../gen/grpc/support/v1/admin_pb.ts"
import {Meanings, wordsOf} from "../refusal.ts"

export type LetterEntry =
    | {kind: "mail", at: Timestamp | undefined, text: string}
    | {kind: "reply", at: Timestamp | undefined, text: string, from: string}
    | {kind: "closed", at: Timestamp | undefined}

// The mails, the player's replies and the last close, by time.
export function threadOf(letter: Letter): LetterEntry[] {
    const entries: LetterEntry[] = [
        ...letter.mails.map(mail => ({kind: "mail" as const, at: mail.sentAt, text: mail.text})),
        ...letter.replies.map(reply => ({kind: "reply" as const, at: reply.receivedAt, text: reply.text, from: reply.from})),
        ...(letter.closedAt === undefined ? [] : [{kind: "closed" as const, at: letter.closedAt}]),
    ]
    return entries.sort((a, b) => msOf(a.at) - msOf(b.at))
}

// As the server's open_only reads it: the player replied since the last mail and the last close.
export function waitsOf(letter: Letter): boolean {
    const replied = msOf(letter.replies.at(-1)?.receivedAt)
    return replied > msOf(letter.mails.at(-1)?.sentAt) && replied > msOf(letter.closedAt)
}

// The letter about a submission, to its founder, or one more mail in a letter.
export type Writing = {letterId: string} | {submissionId: string, accountId: string, subject: string}

export async function writeLetter(support: SupportAdmin, writing: Writing, text: string): Promise<Letter> {
    const request = "letterId" in writing
        ? {letterId: writing.letterId, text: text.trim()}
        : {submissionId: writing.submissionId, accountId: writing.accountId, subject: writing.subject.trim(), text: text.trim()}
    const {letter} = await support.writeLetter(request)
    if (letter === undefined) throw new ConnectError("the server answered no letter", Code.Internal)
    return letter
}

export async function closeLetter(support: SupportAdmin, letter: Letter): Promise<Letter> {
    const {closedAt} = await support.closeLetter({letterId: letter.id})
    return new Letter({...letter, closedAt})
}

const GONE = "This letter is no longer kept."

const MEANINGS: Record<"write" | "close", Meanings> = {
    write: {
        notFound: GONE,
        failedPrecondition: "No mail can go: the player has no verified address, or the letter is over 90 days old.",
        unavailable: "The mail did not go, and nothing was kept. Send it again.",
        alreadyExists: "This submission has its letter already. Reload the page to write in it.",
    },
    close: {notFound: GONE},
}

export function letterRefusalOf(action: "write" | "close", err: unknown): string {
    return wordsOf(err, MEANINGS[action])
}

function msOf(at: Timestamp | undefined): number {
    return at?.toDate().getTime() ?? 0
}
