import {
    ChatAnnouncement,
    ChatBlockedError,
    ChatHistory,
    ChatHistoryGetter,
    ChatListener,
    ChatMessage,
    ChatMessageGoneError,
    ChatRateLimitedError,
    ChatReactor,
    ChatRejectedError,
    ChatSender,
    countRunes,
    MAX_TEXT_LENGTH,
    OutgoingMessage,
    OutgoingReaction,
    Reaction,
    ReactionCount,
    ReactionsChange,
} from "./chat.ts";
import {v4 as UUIDv4} from 'uuid';

const MESSAGES_PER_SECOND = 0.33
const MESSAGE_BURST = 5

export type FakeChatBackendOptions = {
    blocked?: boolean
    chatterIntervalMs?: number
}

// Players with a username and guests, as the server names them. Ana is an admin.
const CHATTERS = [
    {name: "Ana", country: "fr", admin: true, text: "who keeps taking Brittany"},
    {name: "guest_91aa3d", country: "de", admin: false, text: "we hold the north 💪"},
    {name: "kiran_07", country: "in", admin: false, text: "gm everyone"},
    {name: "guest_aa1290", country: "jp", admin: false, text: "the pacific is ours"},
]

/**
 * The guest this browser posts as. There is no account here, so no token names
 * a username: every message from this browser is a guest's, under the one code.
 */
export const OWN_GUEST_NAME = "guest_c0ffee"

// Who reacts from this browser: like the server, one reactor per account.
const ME = "me"

// What the bots react with, now and then.
const BOT_REACTIONS = [Reaction.LAUGH, Reaction.CLOWN, Reaction.SKULL, Reaction.FIRE, Reaction.EARTH]

type Listener = {
    message: (message: ChatMessage) => void
    reactions?: (change: ReactionsChange) => void
    announcement?: (announcement: ChatAnnouncement) => void
}

export class FakeChatBackend implements ChatSender, ChatHistoryGetter, ChatListener, ChatReactor {
    private readonly messages: ChatMessage[] = []
    private readonly announcements: ChatAnnouncement[] = []
    // Message id → reaction → who gave it, in the order each reaction first appeared.
    private readonly reactions = new Map<string, Map<Reaction, Set<string>>>()
    // Message id → how many times its reactions changed, as the server counts them.
    private readonly versions = new Map<string, number>()
    private readonly listeners = new Map<string, Listener>()
    private readonly timers: ReturnType<typeof setInterval>[] = []
    private readonly blocked: boolean
    private tokens = MESSAGE_BURST
    private lastRefillMs = Date.now()
    private nextChatter = 0

    constructor(options: FakeChatBackendOptions = {}) {
        this.blocked = options.blocked ?? false

        CHATTERS.forEach((chatter, index) => {
            this.publish({
                id: UUIDv4(),
                sentAt: Date.now() - (CHATTERS.length - index) * 60_000,
                authorName: chatter.name,
                authorAdmin: chatter.admin,
                countryCode: chatter.country,
                text: chatter.text,
                reactions: [],
                reactionsVersion: 0,
            })
        })

        this.timers.push(setInterval(() => {
            const chatter = CHATTERS[this.nextChatter % CHATTERS.length]
            this.nextChatter++
            this.publish({
                id: UUIDv4(),
                sentAt: Date.now(),
                authorName: chatter.name,
                authorAdmin: chatter.admin,
                countryCode: chatter.country,
                text: `${chatter.text} (${this.nextChatter})`,
                reactions: [],
                reactionsVersion: 0,
            })

            const target = this.messages[Math.floor(Math.random() * this.messages.length)]
            const reaction = BOT_REACTIONS[this.nextChatter % BOT_REACTIONS.length]
            this.give(target.id, reaction, chatter.name, true)
        }, options.chatterIntervalMs ?? 8000))
    }

    public close() {
        this.timers.forEach(clearInterval)
        this.timers.length = 0
        this.listeners.clear()
    }

    public async sendMessage(message: OutgoingMessage): Promise<ChatMessage> {
        if (this.blocked) throw new ChatBlockedError()

        const text = message.text.trim()
        if (text === "" || countRunes(text) > MAX_TEXT_LENGTH) throw new ChatRejectedError()

        if (!this.allow()) throw new ChatRateLimitedError()

        const sent: ChatMessage = {
            id: UUIDv4(),
            sentAt: Date.now(),
            authorName: OWN_GUEST_NAME,
            authorAdmin: false,
            countryCode: message.countryCode,
            text,
            reactions: [],
            reactionsVersion: 0,
        }

        this.publish(sent)
        return sent
    }

    public async react(reaction: OutgoingReaction): Promise<ReactionsChange> {
        if (this.blocked) throw new ChatBlockedError()
        if (!this.messages.some(message => message.id === reaction.messageId)) throw new ChatMessageGoneError()

        this.give(reaction.messageId, reaction.reaction, ME, reaction.on)
        return this.changeOf(reaction.messageId, ME)
    }

    public async getHistory(signal?: AbortSignal): Promise<ChatHistory> {
        signal?.throwIfAborted()
        return {
            messages: this.messages.map(message => ({
                ...message,
                reactions: this.tally(message.id, ME),
                reactionsVersion: this.versions.get(message.id) ?? 0,
            })),
            announcements: [...this.announcements],
        }
    }

    /**
     * A bomb landed, as the server's chat hears it from the planet. The fake
     * has no borders, so it never names the ground that was hit.
     */
    public announceBomb(drop: {countryId: string, tile: number | undefined, cleared: readonly number[]}) {
        const announcement: ChatAnnouncement = {
            kind: "bomb",
            id: UUIDv4(),
            announcedAt: Date.now(),
            country: drop.countryId,
            tile: drop.tile,
            cleared: drop.cleared.length,
        }
        this.announcements.push(announcement)
        this.listeners.forEach(listener => listener.announcement?.(announcement))
    }

    public listenForMessages(
        callback: (message: ChatMessage) => void,
        onReactions?: (change: ReactionsChange) => void,
        onAnnouncement?: (announcement: ChatAnnouncement) => void,
    ): () => void {
        const id = UUIDv4()
        this.listeners.set(id, {message: callback, reactions: onReactions, announcement: onAnnouncement})
        return () => {
            this.listeners.delete(id)
        }
    }

    private publish(message: ChatMessage) {
        this.messages.push(message)
        this.listeners.forEach(listener => listener.message(message))
    }

    private give(messageId: string, reaction: Reaction, reactor: string, on: boolean) {
        const given = this.reactions.get(messageId) ?? new Map<Reaction, Set<string>>()
        this.reactions.set(messageId, given)
        const reactors = given.get(reaction) ?? new Set<string>()
        if (reactors.has(reactor) === on) return

        if (on) reactors.add(reactor)
        else reactors.delete(reactor)
        if (reactors.size === 0) given.delete(reaction)
        else given.set(reaction, reactors)

        this.versions.set(messageId, (this.versions.get(messageId) ?? 0) + 1)
        const change = this.changeOf(messageId, undefined)
        this.listeners.forEach(listener => listener.reactions?.(change))
    }

    private changeOf(messageId: string, viewer: string | undefined): ReactionsChange {
        return {messageId, reactions: this.tally(messageId, viewer), version: this.versions.get(messageId) ?? 0}
    }

    private tally(messageId: string, viewer: string | undefined): ReactionCount[] {
        return [...(this.reactions.get(messageId) ?? new Map<Reaction, Set<string>>())]
            .map(([reaction, reactors]) => ({
                reaction,
                count: reactors.size,
                mine: viewer !== undefined && reactors.has(viewer),
            }))
    }

    private allow(): boolean {
        const now = Date.now()
        this.tokens = Math.min(
            MESSAGE_BURST,
            this.tokens + ((now - this.lastRefillMs) / 1000) * MESSAGES_PER_SECOND,
        )
        this.lastRefillMs = now

        if (this.tokens < 1) return false
        this.tokens -= 1
        return true
    }
}
