import {
    ChatAnnouncement,
    ChatBlockedError,
    ChatHistory,
    ChatHistoryGetter,
    ChatListener,
    ChatMessage,
    ChatMessageGoneError,
    ChatNoSessionError,
    ChatRateLimitedError,
    ChatReactor,
    ChatRejectedError,
    ChatSender,
    OutgoingMessage,
    OutgoingReaction,
    ReactionCount,
    ReactionsChange,
} from "./chat.ts";
import {
    Announcement as AnnouncementPb,
    ChatEvent,
    ChatMessage as ChatMessagePb,
    ReactionCount as ReactionCountPb,
} from "../gen/grpc/chat/v1/chat_pb.ts";
import {ChatService} from "../gen/grpc/chat/v1/chat_connect.ts";
import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect";
import {createConnectTransport} from "@connectrpc/connect-web";
import {Config, NO_TIMEOUT, openStream, retrying} from "./transport.ts";
import {SESSION_HEADER, SessionProvider} from "./session.ts";

export function newChatServiceClient(config: Config): PromiseClient<typeof ChatService> {
    return createPromiseClient(ChatService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class ChatServiceBackend implements ChatSender, ChatHistoryGetter, ChatListener, ChatReactor {
    constructor(
        private client: PromiseClient<typeof ChatService>,
        private readonly session: SessionProvider,
    ) {
    }

    public async sendMessage(message: OutgoingMessage): Promise<ChatMessage> {
        try {
            const res = await this.authenticated((headers) => this.client.sendMessage({
                authorId: message.authorId,
                countryId: message.countryCode,
                text: message.text,
            }, {headers}))

            if (!res.message) throw new ChatRejectedError()

            return decodedMessage(res.message)
        } catch (e) {
            throw translate(e)
        }
    }

    public async react(reaction: OutgoingReaction): Promise<ReactionsChange> {
        try {
            const res = await this.authenticated((headers) => this.client.react({
                messageId: reaction.messageId,
                reaction: reaction.reaction,
                on: reaction.on,
            }, {headers}))

            return {
                messageId: reaction.messageId,
                reactions: res.reactions.map(decodedCount),
                version: Number(res.version),
            }
        } catch (e) {
            throw translate(e)
        }
    }

    /**
     * The server names the sender by the click token, guests included, so a
     * token is minted when none is held, as a click does. One the server
     * refuses is dropped and the call made once more with a fresh one: a
     * refused call posted nothing, so this cannot post twice.
     */
    private async authenticated<T>(call: (headers: Headers) => Promise<T>): Promise<T> {
        try {
            return await call(await this.headers())
        } catch (e) {
            if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) throw e

            this.session.invalidate()
            return await call(await this.headers())
        }
    }

    private async headers(): Promise<Headers> {
        let token: string | undefined
        try {
            token = await this.session.token()
        } catch (e) {
            throw new ChatNoSessionError({cause: e})
        }

        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)
        return headers
    }

    /**
     * Sends the token already held, never a fresh one, so the server can mark
     * the caller's own reactions: loading the chat is not worth a mint.
     */
    public async getHistory(signal?: AbortSignal): Promise<ChatHistory> {
        const headers = new Headers()
        const held = this.session.held()
        if (held) headers.set(SESSION_HEADER, held)

        try {
            const res = await retrying(
                () => this.client.getHistory({}, {signal, headers}),
                "chat history",
                signal,
            )

            return {
                messages: res.messages.map(decodedMessage),
                announcements: res.announcements.flatMap(announcement => decodedAnnouncement(announcement) ?? []),
            }
        } catch (e) {
            throw translate(e)
        }
    }

    public listenForMessages(
        callback: (message: ChatMessage) => void,
        onReactions?: (change: ReactionsChange) => void,
        onAnnouncement?: (announcement: ChatAnnouncement) => void,
    ): () => void {
        return openStream(
            (signal) => this.client.listenForEvents({}, {signal, timeoutMs: NO_TIMEOUT}),
            (event) => {
                const message = messageOf(event)
                if (message) callback(message)
                const change = reactionsOf(event)
                if (change) onReactions?.(change)
                const announcement = announcementOf(event)
                if (announcement) onAnnouncement?.(announcement)
            },
            "chat",
        )
    }
}

function translate(e: unknown): unknown {
    if (!(e instanceof ConnectError)) return e

    switch (e.code) {
        case Code.ResourceExhausted:
            return new ChatRateLimitedError({cause: e})
        case Code.PermissionDenied:
            return new ChatBlockedError({cause: e})
        case Code.InvalidArgument:
            return new ChatRejectedError({cause: e})
        case Code.NotFound:
            return new ChatMessageGoneError({cause: e})
        case Code.Unauthenticated:
            return new ChatNoSessionError({cause: e})
        default:
            return e
    }
}

/**
 * Anything that is not a message is dropped, heartbeats included — as is an
 * event case this build does not know, which reads as an unset `oneof`.
 */
export function messageOf(event: ChatEvent): ChatMessage | undefined {
    if (event.event.case !== "message") return undefined

    return decodedMessage(event.event.value)
}

export function decodedMessage(message: ChatMessagePb): ChatMessage {
    return {
        id: message.id,
        sentAt: Number(message.sentAtUnixMs),
        authorName: message.authorName,
        authorAdmin: message.authorAdmin,
        countryCode: message.countryId,
        text: message.text,
        reactions: message.reactions.map(decodedCount),
        reactionsVersion: Number(message.reactionsVersion),
    }
}

export function reactionsOf(event: ChatEvent): ReactionsChange | undefined {
    if (event.event.case !== "reactions") return undefined

    return {
        messageId: event.event.value.messageId,
        reactions: event.event.value.reactions.map(decodedCount),
        version: Number(event.event.value.version),
    }
}

export function announcementOf(event: ChatEvent): ChatAnnouncement | undefined {
    if (event.event.case !== "announcement") return undefined

    return decodedAnnouncement(event.event.value)
}

/**
 * An announcement this build can write a line for, or undefined: a kind it
 * does not know, or a payload that is not the kind's, is not shown.
 */
export function decodedAnnouncement(announcement: AnnouncementPb): ChatAnnouncement | undefined {
    let payload: unknown
    try {
        payload = JSON.parse(announcement.payload)
    } catch {
        return undefined
    }
    if (typeof payload !== "object" || payload === null) return undefined
    const values = payload as Record<string, unknown>

    switch (announcement.kind) {
        case "bomb":
            if (typeof values.country !== "string" || values.country === "") return undefined
            return {
                kind: "bomb",
                id: announcement.id,
                announcedAt: Number(announcement.announcedAtUnixMs),
                country: values.country,
                ground: typeof values.ground === "string" && values.ground !== "" ? values.ground : undefined,
                tile: typeof values.tile === "number" && values.tile > 0 ? values.tile : undefined,
                cleared: typeof values.cleared === "number" ? values.cleared : 0,
            }
        default:
            return undefined
    }
}

function decodedCount(count: ReactionCountPb): ReactionCount {
    return {reaction: count.reaction, count: count.count, mine: count.mine, reactors: count.reactors}
}
