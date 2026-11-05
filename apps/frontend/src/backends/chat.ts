import {Reaction} from "../gen/grpc/chat/v1/chat_pb.ts";
import {NameColor} from "../gen/grpc/player/v1/color_pb.ts";

export {Reaction}

export const MAX_TEXT_LENGTH = 280

export const GUEST_PREFIX = "guest_"

export function countRunes(value: string): number {
    return [...value].length
}

// At the address KLIPY gave: its terms forbid rewriting it.
export type Gif = {
    url: string
    width: number
    height: number
}

export type ChatMessage = {
    id: string
    sentAt: number
    authorAccountId: string
    authorName: string
    authorAdmin: boolean
    authorColor: NameColor
    countryCode: string
    text: string
    reactions: ReactionCount[]
    reactionsVersion: number
    replyTo?: ChatQuote
    removed?: boolean
    // A message shows a text or a GIF, never both.
    gif?: Gif
}

export type ChatQuote = {
    messageId: string
    authorName: string
    authorColor: NameColor
    text: string
    removed?: boolean
    gif?: Gif
}

export type ChatAnnouncement = BombAnnouncement | MuteAnnouncement | RoundAnnouncement | FortifyAnnouncement | ChampionAnnouncement

export type BombAnnouncement = {
    kind: "bomb"
    id: string
    announcedAt: number
    country: string
    ground?: string
    landmass?: number
    tile?: number
    cleared: number
}

export type FortifyAnnouncement = {
    kind: "fortify"
    id: string
    announcedAt: number
    country: string
    ground: string
    landmass: number
    tiles: number
}

export type MuteAnnouncement = {
    kind: "mute"
    id: string
    announcedAt: number
    name: string
    // What the mute takes away. A line kept before kinds is the whole chat's.
    muteKind: MuteKind
    seconds: number
}

export type PodiumPlace = {
    country: string
    rank: number
    points: number
}

export type RoundAnnouncement = {
    kind: "round"
    id: string
    announcedAt: number
    number: number
    finale: boolean
    own: boolean
    times: number
    podium: PodiumPlace[]
}

export type ChampionAnnouncement = {
    kind: "champion"
    id: string
    announcedAt: number
    season: number
    countries: string[]
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
    replyTo?: string
    gif?: Gif
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
        onResumed?: () => void,
        onRemoved?: (messageId: string) => void,
        onLeft?: () => void,
    ): () => void
}

export interface ChatReactor {
    react(reaction: OutgoingReaction): Promise<ReactionsChange>
}

export interface ChatSeenMarker {
    markSeen(until: number): Promise<void>
}

export type ChatBackend = ChatSender & ChatHistoryGetter & ChatListener & ChatReactor & ChatSeenMarker

export interface ChatFlagRooms {
    flagRoom(flag: string): ChatBackend
}

export type ChatRooms = ChatBackend & ChatFlagRooms

export class ChatNotInRoomError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("this player does not play for the room's flag", options)
        this.name = "ChatNotInRoomError"
    }
}

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

// What a mute takes away: the whole chat, or a post with a GIF alone.
export type MuteKind = 'chat' | 'gifs'

export class ChatMutedError extends Error {
    readonly until: number
    readonly kind: MuteKind

    constructor(until: number, kind: MuteKind = 'chat', options?: {cause?: unknown}) {
        super("muted in the chat", options)
        this.name = "ChatMutedError"
        this.until = until
        this.kind = kind
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
