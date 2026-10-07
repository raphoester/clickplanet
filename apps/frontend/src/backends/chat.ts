import {Reaction} from "../gen/grpc/chat/v1/chat_pb.ts";
import {NameColor} from "../gen/grpc/player/v1/color_pb.ts";
import {PlayerTitle} from "./title.ts";

export {Reaction}

export const MAX_TEXT_LENGTH = 280

export const GUEST_PREFIX = "guest_"

export function countRunes(value: string): number {
    return [...value].length
}

export type ChatMessage = {
    id: string
    sentAt: number
    authorName: string
    authorAdmin: boolean
    authorColor: NameColor
    authorStreak: number
    authorTitle?: PlayerTitle
    countryCode: string
    text: string
    reactions: ReactionCount[]
    reactionsVersion: number
}

export type ChatAnnouncement = BombAnnouncement | MuteAnnouncement | LeadChangedAnnouncement | SeasonWonAnnouncement

export type BombAnnouncement = {
    kind: "bomb"
    id: string
    announcedAt: number
    country: string
    ground?: string
    tile?: number
    cleared: number
}

export type MuteAnnouncement = {
    kind: "mute"
    id: string
    announcedAt: number
    name: string
    seconds: number
}

export type LeadChangedAnnouncement = {
    kind: "leadChanged"
    id: string
    announcedAt: number
    season: number
    leader: string
    passed: string
}

export type SeasonWonAnnouncement = {
    kind: "seasonWon"
    id: string
    announcedAt: number
    season: number
    winner: string
}

export type ChatHistory = {
    messages: ChatMessage[]
    announcements: ChatAnnouncement[]
    seenUntil?: number
}

export type ReactionCount = {
    reaction: Reaction
    count: number
    mine: boolean
    reactors: string[]
}

export type ReactionsChange = {
    messageId: string
    reactions: ReactionCount[]
    version: number
}

export type OutgoingReaction = {
    messageId: string
    reaction: Reaction
    on: boolean
}

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
    react(reaction: OutgoingReaction): Promise<ReactionsChange>
}

export interface ChatSeenMarker {
    markSeen(until: number): Promise<void>
}

export type ChatBackend = ChatSender & ChatHistoryGetter & ChatListener & ChatReactor & ChatSeenMarker

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

export class ChatMutedError extends Error {
    readonly until: number

    constructor(until: number, options?: {cause?: unknown}) {
        super("muted in the chat", options)
        this.name = "ChatMutedError"
        this.until = until
    }
}

export class ChatMessageGoneError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("the message is gone", options)
        this.name = "ChatMessageGoneError"
    }
}

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
