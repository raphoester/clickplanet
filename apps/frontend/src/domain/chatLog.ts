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

function byArrival(a: ChatMessage, b: ChatMessage): number {
    if (a.sentAt !== b.sentAt) return a.sentAt - b.sentAt
    return a.id < b.id ? -1 : a.id > b.id ? 1 : 0
}
