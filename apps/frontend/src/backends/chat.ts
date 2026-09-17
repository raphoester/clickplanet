export const MAX_TEXT_LENGTH = 280
/** A guest's typed name, before the server puts `GUEST_PREFIX` in front of it. */
export const MAX_NAME_LENGTH = 24

/**
 * What the server puts before every guest's name. No username starts with it,
 * so a guest cannot pass for a player.
 */
export const GUEST_PREFIX = "guest_"

export function countRunes(value: string): number {
    return [...value].length
}

/** The name the log shows for a guest who typed `name`. */
export function guestName(name: string): string {
    return GUEST_PREFIX + name
}

export type ChatMessage = {
    id: string
    sentAt: number
    /** A username, or `GUEST_PREFIX` and a guest's name. */
    authorName: string
    authorTag: string
    countryCode: string
    text: string
}

export type OutgoingMessage = {
    /** A guest's name. The server does not read it for a player with a username. */
    authorName: string
    authorId: string
    countryCode: string
    text: string
    /**
     * Sent by a player with a username: the click token goes along, and the
     * server posts under the username. A guest sends no token, so chatting
     * never mints a session.
     */
    asAccount: boolean
}

export interface ChatSender {
    sendMessage(message: OutgoingMessage): Promise<ChatMessage>
}

export interface ChatHistoryGetter {
    getHistory(signal?: AbortSignal): Promise<ChatMessage[]>
}

export interface ChatListener {
    listenForMessages(callback: (message: ChatMessage) => void): () => void
}

export type ChatBackend = ChatSender & ChatHistoryGetter & ChatListener

export class ChatRateLimitedError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("too many messages", options)
        this.name = "ChatRateLimitedError"
    }
}

export class ChatBlockedError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("this address is not allowed to post", options)
        this.name = "ChatBlockedError"
    }
}

export class ChatRejectedError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("the message was refused", options)
        this.name = "ChatRejectedError"
    }
}
