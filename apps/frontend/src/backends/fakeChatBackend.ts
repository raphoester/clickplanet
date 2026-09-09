import {
    ChatBlockedError,
    ChatHistoryGetter,
    ChatListener,
    ChatMessage,
    ChatRateLimitedError,
    ChatRejectedError,
    ChatSender,
    ChatUnavailableError,
    countRunes,
    MAX_NAME_LENGTH,
    MAX_TEXT_LENGTH,
    OutgoingMessage,
} from "./chat.ts";
import {v4 as UUIDv4} from 'uuid';

const MESSAGES_PER_SECOND = 0.33
const MESSAGE_BURST = 5

export type FakeChatBackendOptions = {
    blocked?: boolean
    unavailable?: boolean
    chatterIntervalMs?: number
}

const CHATTERS = [
    {name: "Ana", tag: "4f2ca1", country: "fr", text: "who keeps taking Brittany"},
    {name: "Bo", tag: "91aa3d", country: "de", text: "we hold the north 💪"},
    {name: "Kiran", tag: "0c77e2", country: "in", text: "gm everyone"},
    {name: "Yuki", tag: "aa1290", country: "jp", text: "the pacific is ours"},
]

export class FakeChatBackend implements ChatSender, ChatHistoryGetter, ChatListener {
    private readonly messages: ChatMessage[] = []
    private readonly listeners = new Map<string, (message: ChatMessage) => void>()
    private readonly timers: ReturnType<typeof setInterval>[] = []
    private readonly blocked: boolean
    private readonly unavailable: boolean
    private tokens = MESSAGE_BURST
    private lastRefillMs = Date.now()
    private nextChatter = 0

    constructor(options: FakeChatBackendOptions = {}) {
        this.blocked = options.blocked ?? false
        this.unavailable = options.unavailable ?? false

        CHATTERS.forEach((chatter, index) => {
            this.publish({
                id: UUIDv4(),
                sentAt: Date.now() - (CHATTERS.length - index) * 60_000,
                authorName: chatter.name,
                authorTag: chatter.tag,
                countryCode: chatter.country,
                text: chatter.text,
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
                countryCode: chatter.country,
                text: `${chatter.text} (${this.nextChatter})`,
            })
        }, options.chatterIntervalMs ?? 8000))
    }

    public close() {
        this.timers.forEach(clearInterval)
        this.timers.length = 0
        this.listeners.clear()
    }

    public async sendMessage(message: OutgoingMessage): Promise<ChatMessage> {
        if (this.unavailable) throw new ChatUnavailableError()
        if (this.blocked) throw new ChatBlockedError()

        const text = message.text.trim()
        const name = message.authorName.trim()
        if (text === "" || countRunes(text) > MAX_TEXT_LENGTH) throw new ChatRejectedError()
        if (name === "" || countRunes(name) > MAX_NAME_LENGTH) throw new ChatRejectedError()

        if (!this.allow()) throw new ChatRateLimitedError()

        const sent: ChatMessage = {
            id: UUIDv4(),
            sentAt: Date.now(),
            authorName: name,
            authorTag: "c0ffee",
            countryCode: message.countryCode,
            text,
        }

        this.publish(sent)
        return sent
    }

    public async getHistory(signal?: AbortSignal): Promise<ChatMessage[]> {
        signal?.throwIfAborted()
        if (this.unavailable) throw new ChatUnavailableError()
        return [...this.messages]
    }

    public listenForMessages(callback: (message: ChatMessage) => void): () => void {
        const id = UUIDv4()
        this.listeners.set(id, callback)
        return () => {
            this.listeners.delete(id)
        }
    }

    private publish(message: ChatMessage) {
        this.messages.push(message)
        this.listeners.forEach(listener => listener(message))
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
