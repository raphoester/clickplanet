export const MAX_TEXT_LENGTH = 280
export const MAX_NAME_LENGTH = 24

export function countRunes(value: string): number {
    return [...value].length
}

export type ChatMessage = {
    id: string
    sentAt: number
    authorName: string
    authorTag: string
    countryCode: string
    text: string
    /**
     * The author was banned from the chat, so `text` is empty and the server
     * will never send it again. Everything else about the line is still here,
     * which is the point: the chat reads as a conversation with one turn
     * blanked rather than losing turns out of it.
     */
    redacted: boolean
}

/**
 * One frame of the live feed, mirroring the `ChatEvent` oneof on the wire. A
 * redaction is what an operator's ban sends, so a banned member's text leaves
 * every open tab without anyone reloading.
 */
export type ChatFeedEvent =
    | {kind: "message", message: ChatMessage}
    | {kind: "redacted", authorTag: string}

export type OutgoingMessage = {
    authorName: string
    authorId: string
    countryCode: string
    text: string
}

export interface ChatSender {
    sendMessage(message: OutgoingMessage): Promise<ChatMessage>
}

export interface ChatHistoryGetter {
    getHistory(signal?: AbortSignal): Promise<ChatMessage[]>
}

export interface ChatListener {
    listenForEvents(callback: (event: ChatFeedEvent) => void): () => void
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
