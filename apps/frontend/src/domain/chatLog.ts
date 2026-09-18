import type {ChatAnnouncement, ChatMessage} from "../backends/chat.ts";

export const CHAT_LOG_LIMIT = 200

/**
 * Keeps each announcement once, oldest first, and the newest `limit` of them.
 * Announcements are held apart from the messages, so a burst of bombs never
 * pushes a message out of the log, and nothing counts one as unread.
 */
export function addAnnouncements(
    log: readonly ChatAnnouncement[],
    incoming: readonly ChatAnnouncement[],
    limit: number = CHAT_LOG_LIMIT,
): ChatAnnouncement[] {
    const known = new Set(log.map(announcement => announcement.id))
    const added = incoming.filter(announcement => {
        if (known.has(announcement.id)) return false
        known.add(announcement.id)
        return true
    })

    if (added.length === 0) return log as ChatAnnouncement[]

    const merged = [...log, ...added].sort((a, b) => a.announcedAt - b.announcedAt || compareIds(a.id, b.id))
    return merged.length > limit ? merged.slice(merged.length - limit) : merged
}

/** One row of the log: a message, or an announcement between messages. */
export type ChatLogEntry =
    | {kind: "message", message: ChatMessage}
    | {kind: "announcement", announcement: ChatAnnouncement}

/** Messages and announcements in one list by time. A message goes first when both happened at once. */
export function interleave(
    messages: readonly ChatMessage[],
    announcements: readonly ChatAnnouncement[],
): ChatLogEntry[] {
    const entries: ChatLogEntry[] = []

    let a = 0
    for (const message of messages) {
        while (a < announcements.length && announcements[a].announcedAt < message.sentAt) {
            entries.push({kind: "announcement", announcement: announcements[a++]})
        }
        entries.push({kind: "message", message})
    }
    while (a < announcements.length) entries.push({kind: "announcement", announcement: announcements[a++]})

    return entries
}

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
 * The name the server gave the latest message this client sent, of those still
 * in the log: a guest's name is the server's pick, and this is how the client
 * learns it.
 */
export function nameSentUnder(log: readonly ChatMessage[], sent: ReadonlySet<string>): string | undefined {
    for (let i = log.length - 1; i >= 0; i--) {
        if (sent.has(log[i].id)) return log[i].authorName
    }
    return undefined
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
    if (previous.authorName !== message.authorName) return true
    return message.sentAt - previous.sentAt > window
}

function byArrival(a: ChatMessage, b: ChatMessage): number {
    if (a.sentAt !== b.sentAt) return a.sentAt - b.sentAt
    return compareIds(a.id, b.id)
}

function compareIds(a: string, b: string): number {
    return a < b ? -1 : a > b ? 1 : 0
}
