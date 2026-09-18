import {
    ChatBlockedError,
    ChatHistoryGetter,
    ChatListener,
    ChatMessage,
    ChatMessageGoneError,
    ChatRateLimitedError,
    ChatReactor,
    ChatRejectedError,
    ChatSender,
    countRunes,
    guestName,
    MAX_NAME_LENGTH,
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
    {name: "Ana", tag: "4f2ca1", country: "fr", admin: true, text: "who keeps taking Brittany"},
    {name: guestName("Bo"), tag: "91aa3d", country: "de", admin: false, text: "we hold the north 💪"},
    {name: "kiran_07", tag: "0c77e2", country: "in", admin: false, text: "gm everyone"},
    {name: guestName("Yuki"), tag: "aa1290", country: "jp", admin: false, text: "the pacific is ours"},
]

// Who reacts from this browser: like the server's guest, one reactor per address.
const ME = "me"

// What the bots react with, now and then.
const BOT_REACTIONS = [Reaction.LAUGH, Reaction.CLOWN, Reaction.SKULL, Reaction.FIRE, Reaction.EARTH]

type Listener = {
    message: (message: ChatMessage) => void
    reactions?: (change: ReactionsChange) => void
}

export class FakeChatBackend implements ChatSender, ChatHistoryGetter, ChatListener, ChatReactor {
    private readonly messages: ChatMessage[] = []
    // Message id → reaction → who gave it, in the order each reaction first appeared.
    private readonly reactions = new Map<string, Map<Reaction, Set<string>>>()
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
                authorTag: chatter.tag,
                authorAdmin: chatter.admin,
                countryCode: chatter.country,
                text: chatter.text,
                reactions: [],
            })
        })

        this.timers.push(setInterval(() => {
            const chatter = CHATTERS[this.nextChatter % CHATTERS.length]
            this.nextChatter++
            this.publish({
                id: UUIDv4(),
                sentAt: Date.now(),
                authorName: chatter.name,
                authorTag: chatter.tag,
                authorAdmin: chatter.admin,
                countryCode: chatter.country,
                text: `${chatter.text} (${this.nextChatter})`,
                reactions: [],
            })

            const target = this.messages[Math.floor(Math.random() * this.messages.length)]
            const reaction = BOT_REACTIONS[this.nextChatter % BOT_REACTIONS.length]
            this.give(target.id, reaction, chatter.tag, true)
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
        const name = message.authorName.trim()
        if (text === "" || countRunes(text) > MAX_TEXT_LENGTH) throw new ChatRejectedError()
        if (name === "" || countRunes(name) > MAX_NAME_LENGTH) throw new ChatRejectedError()

        if (!this.allow()) throw new ChatRateLimitedError()

        // There is no account here, so no token names a username: like the
        // server, every message from this browser is a guest's.
        const sent: ChatMessage = {
            id: UUIDv4(),
            sentAt: Date.now(),
            authorName: guestName(name),
            authorTag: "c0ffee",
            authorAdmin: false,
            countryCode: message.countryCode,
            text,
            reactions: [],
        }

        this.publish(sent)
        return sent
    }

    public async react(reaction: OutgoingReaction): Promise<ReactionCount[]> {
        if (this.blocked) throw new ChatBlockedError()
        if (!this.messages.some(message => message.id === reaction.messageId)) throw new ChatMessageGoneError()

        this.give(reaction.messageId, reaction.reaction, ME, reaction.on)
        return this.tally(reaction.messageId, ME)
    }

    public async getHistory(signal?: AbortSignal): Promise<ChatMessage[]> {
        signal?.throwIfAborted()
        return this.messages.map(message => ({...message, reactions: this.tally(message.id, ME)}))
    }

    public listenForMessages(
        callback: (message: ChatMessage) => void,
        onReactions?: (change: ReactionsChange) => void,
    ): () => void {
        const id = UUIDv4()
        this.listeners.set(id, {message: callback, reactions: onReactions})
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

        const change = {messageId, reactions: this.tally(messageId, undefined)}
        this.listeners.forEach(listener => listener.reactions?.(change))
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
