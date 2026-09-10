import type {ChatMessage} from "../backends/chat.ts";

export const CHAT_LOG_LIMIT = 200

export function addMessages(
    log: readonly ChatMessage[],
    incoming: readonly ChatMessage[],
    limit: number = CHAT_LOG_LIMIT,
): ChatMessage[] {
    const known = new Set(log.map(message => message.id))
    const added = incoming.filter(message => {
        if (known.has(message.id)) return false
        known.add(message.id)
        return true
    })

    if (added.length === 0) return log as ChatMessage[]

    const merged = [...log, ...added].sort(byArrival)
    return merged.length > limit ? merged.slice(merged.length - limit) : merged
}

export function unreadSince(log: readonly ChatMessage[], lastSeenId: string | undefined): number {
    if (lastSeenId === undefined) return log.length

    const index = log.findIndex(message => message.id === lastSeenId)
    return index === -1 ? log.length : log.length - 1 - index
}

/**
 * The ids `unreadSince` counted, for highlighting them once they are on screen.
 */
export function idsSince(log: readonly ChatMessage[], lastSeenId: string | undefined): string[] {
    const count = unreadSince(log, lastSeenId)
    return count === 0 ? [] : log.slice(log.length - count).map(message => message.id)
}

/**
 * How long a quiet gap has to be before the same author's next message starts a
 * new group rather than joining the run above it.
 */
export const GROUP_WINDOW_MS = 4 * 60_000

/**
 * Whether a message opens a group — the one message in a run that says who is
 * talking and when. A run is one author speaking without a long pause; the rest
 * of it is bubbles alone, which is what a chat looks like.
 */
export function startsGroup(
    previous: ChatMessage | undefined,
    message: ChatMessage,
    window: number = GROUP_WINDOW_MS,
): boolean {
    if (previous === undefined) return true
    if (previous.authorTag !== message.authorTag) return true
    if (previous.authorName !== message.authorName) return true
    return message.sentAt - previous.sentAt > window
}

function byArrival(a: ChatMessage, b: ChatMessage): number {
    if (a.sentAt !== b.sentAt) return a.sentAt - b.sentAt
    return a.id < b.id ? -1 : a.id > b.id ? 1 : 0
}
