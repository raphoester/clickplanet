import {KeptMessage} from "../gen/grpc/chat/v1/admin_pb.ts"
import {Page} from "./chatPages.ts"

// How many messages each side of one of theirs a conversation starts with, and how many conversations are built at a
// time.
export const AROUND = 5
export const CONVERSATIONS = 3

// A stretch of the chat, oldest first, with whether more is kept before it and was sent after it.
export type Window = {lines: KeptMessage[], moreBefore: boolean, moreAfter: boolean}

export type Read = (ask: {beforeMessageId?: string, afterMessageId?: string}, limit: number) => Promise<Page>

// The conversations around their messages not shown yet, newest first, added to those shown, up to `count` more. Their
// messages a conversation holds need none of their own, and a conversation that reaches the one shown after it in
// the chat joins it, so an exchange of three insults is one conversation, not three that overlap.
export async function moreConversations(
    shown: Window[], theirs: KeptMessage[], read: Read, count: number,
): Promise<{shown: Window[], left: KeptMessage[]}> {
    const windows = [...shown]
    let left = withoutShown(theirs, windows)
    let built = 0

    while (left.length > 0 && built < count) {
        const message = left[0]
        const [before, after] = await Promise.all([
            read({beforeMessageId: message.messageId}, AROUND),
            read({afterMessageId: message.messageId}, AROUND),
        ])
        const window = {
            lines: [...before.messages, message, ...after.messages],
            moreBefore: before.more,
            moreAfter: after.more,
        }

        const later = windows.at(-1)
        const joined = later === undefined ? undefined : joinedWith(window, later)
        if (joined === undefined) {
            windows.push(window)
            built++
        } else {
            windows[windows.length - 1] = joined
        }
        left = withoutShown(left, windows)
    }

    return {shown: windows, left}
}

// The earlier window followed by the later one, when they share a line; nothing when a gap lies between them.
function joinedWith(earlier: Window, later: Window): Window | undefined {
    const at = earlier.lines.findIndex(line => line.messageId === later.lines[0]?.messageId)
    if (at < 0) return undefined
    return {lines: [...earlier.lines.slice(0, at), ...later.lines], moreBefore: earlier.moreBefore, moreAfter: later.moreAfter}
}

function withoutShown(theirs: KeptMessage[], windows: Window[]): KeptMessage[] {
    const shown = new Set(windows.flatMap(window => window.lines.map(line => line.messageId)))
    return theirs.filter(message => !shown.has(message.messageId))
}

// The title of the panel that holds what one account said.
export function theirTitleOf(name: string): string {
    return name === "" ? "What they said" : `What ${name} said`
}
