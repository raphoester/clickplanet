import {Reaction} from "../gen/grpc/chat/v1/chat_pb.ts";

export {Reaction}

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
    /** Sent under the username of an admin of the game. Never a guest. */
    authorAdmin: boolean
    countryCode: string
    text: string
    /** In the order each reaction first appeared. */
    reactions: ReactionCount[]
}

export type ReactionCount = {
    reaction: Reaction
    count: number
    /** This player gave it. The stream never knows, so `useChat` keeps it between calls. */
    mine: boolean
}

/** A message's reactions changed: all of them, not the difference. */
export type ReactionsChange = {
    messageId: string
    reactions: ReactionCount[]
}

export type OutgoingReaction = {
    messageId: string
    reaction: Reaction
    /** True puts it on, false takes it off. Asking for what is there changes nothing. */
    on: boolean
    /** As on `OutgoingMessage`: a player reacts as its account, everyone else as its address. */
    asAccount: boolean
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
    listenForMessages(
        callback: (message: ChatMessage) => void,
        onReactions?: (change: ReactionsChange) => void,
    ): () => void
}

export interface ChatReactor {
    /** Answers the message's reactions once this one landed, `mine` included. */
    react(reaction: OutgoingReaction): Promise<ReactionCount[]>
}

export type ChatBackend = ChatSender & ChatHistoryGetter & ChatListener & ChatReactor

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

/** A reaction to a message the server no longer shows. */
export class ChatMessageGoneError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("the message is gone", options)
        this.name = "ChatMessageGoneError"
    }
}

export class ChatRejectedError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("the message was refused", options)
        this.name = "ChatRejectedError"
    }
}
