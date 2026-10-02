import {Reaction} from "../gen/grpc/chat/v1/chat_pb.ts";

export {Reaction}

export const MAX_TEXT_LENGTH = 280

/**
 * What the server puts before every guest's name, the guest's code after it
 * (`guest_a1b2c3`). No username starts with it, so a guest cannot pass for a
 * player.
 */
export const GUEST_PREFIX = "guest_"

export function countRunes(value: string): number {
    return [...value].length
}

export type ChatMessage = {
    id: string
    sentAt: number
    /**
     * A username, or `GUEST_PREFIX` and the guest's code: the server picks it,
     * and no two accounts share one.
     */
    authorName: string
    /** Sent under the username of an admin of the game. Never a guest. */
    authorAdmin: boolean
    countryCode: string
    text: string
    /** In the order each reaction first appeared. */
    reactions: ReactionCount[]
    /** Which state of the reactions this is: a lower one never replaces a higher one. */
    reactionsVersion: number
}

/**
 * A line the chat says on its own, between the messages. Nobody sent it, so it
 * has no author, no reactions and no bubble.
 */
export type ChatAnnouncement = BombAnnouncement

/** A bomb landed somewhere on the planet. */
export type BombAnnouncement = {
    kind: "bomb"
    id: string
    announcedAt: number
    /** The code of the country the bomber played for. */
    country: string
    /** The code of the country whose ground it hit. Absent in the sea, and on no country's ground. */
    ground?: string
    /** The tile it hit. Absent in the sea. */
    tile?: number
    /** How many held tiles it cleared. */
    cleared: number
}

/** What a joining client is shown. Each list is oldest first; the log puts them in one by time. */
export type ChatHistory = {
    messages: ChatMessage[]
    announcements: ChatAnnouncement[]
}

export type ReactionCount = {
    reaction: Reaction
    count: number
    /** This player gave it. The stream never knows, so `useChat` keeps it between calls. */
    mine: boolean
    /**
     * Who gave it, oldest first, each under the name it went by when it
     * reacted. The server cuts the list at a cap, so it can be shorter than
     * `count` — and it is shorter again for a moment after this player
     * reacted, until the server answers with the name. `count` is always how
     * many gave it.
     */
    reactors: string[]
}

/**
 * A message's reactions changed: all of them, not the difference. Frames can
 * arrive out of order, so each carries the version it was read at.
 */
export type ReactionsChange = {
    messageId: string
    reactions: ReactionCount[]
    version: number
}

export type OutgoingReaction = {
    messageId: string
    reaction: Reaction
    /** True puts it on, false takes it off. Asking for what is there changes nothing. */
    on: boolean
}

/**
 * Carries no name: the click token goes along, and the server posts under the
 * name of the account it names.
 */
export type OutgoingMessage = {
    authorId: string
    countryCode: string
    text: string
}

export interface ChatSender {
    sendMessage(message: OutgoingMessage): Promise<ChatMessage>
}

export interface ChatHistoryGetter {
    getHistory(signal?: AbortSignal): Promise<ChatHistory>
}

export interface ChatListener {
    listenForMessages(
        callback: (message: ChatMessage) => void,
        onReactions?: (change: ReactionsChange) => void,
        onAnnouncement?: (announcement: ChatAnnouncement) => void,
    ): () => void
}

export interface ChatReactor {
    /** Answers the message's reactions once this one landed, `mine` included. */
    react(reaction: OutgoingReaction): Promise<ReactionsChange>
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

/**
 * No session could name the sender: the mint failed, or the server refused the
 * token twice. Nothing was posted, and a later try may well work.
 */
export class ChatNoSessionError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("no session to chat with", options)
        this.name = "ChatNoSessionError"
    }
}

export class ChatRejectedError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("the message was refused", options)
        this.name = "ChatRejectedError"
    }
}
