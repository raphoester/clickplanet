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
}

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

export class ChatUnavailableError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("chat is not enabled on this server", options)
        this.name = "ChatUnavailableError"
    }
}
