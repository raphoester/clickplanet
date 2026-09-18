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
    OutgoingMessage,
    OutgoingReaction,
    ReactionCount,
    ReactionsChange,
} from "./chat.ts";
import {
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
        const headers = await this.headersFor(message.asAccount)

        try {
            const res = await this.client.sendMessage({
                authorName: message.authorName,
                authorId: message.authorId,
                countryId: message.countryCode,
                text: message.text,
            }, {headers})

            if (!res.message) throw new ChatRejectedError()

            return decodedMessage(res.message)
        } catch (e) {
            throw translate(e)
        }
    }

    /**
     * The click token, only for a player with a username: a guest sends none,
     * so chatting never mints a session. A token that cannot be had is not a
     * failure — the server reads a message without one as a guest's.
     */
    private async headersFor(asAccount: boolean): Promise<Headers> {
        const headers = new Headers()
        if (!asAccount) return headers

        const token = await this.session.token().catch((e) => {
            console.error("No session for the chat: sending as a guest", e)
            return undefined
        })
        if (token) headers.set(SESSION_HEADER, token)
        return headers
    }

    public async react(reaction: OutgoingReaction): Promise<ReactionCount[]> {
        const headers = await this.headersFor(reaction.asAccount)

        try {
            const res = await this.client.react({
                messageId: reaction.messageId,
                reaction: reaction.reaction,
                on: reaction.on,
            }, {headers})

            return res.reactions.map(decodedCount)
        } catch (e) {
            throw translate(e)
        }
    }

    /**
     * Sends the token already held, never a fresh one, so the server can mark
     * the caller's own reactions: loading the chat is not worth a mint.
     */
    public async getHistory(signal?: AbortSignal): Promise<ChatMessage[]> {
        const headers = new Headers()
        const held = this.session.held()
        if (held) headers.set(SESSION_HEADER, held)

        try {
            const res = await retrying(
                () => this.client.getHistory({}, {signal, headers}),
                "chat history",
                signal,
            )

            return res.messages.map(decodedMessage)
        } catch (e) {
            throw translate(e)
        }
    }

    public listenForMessages(
        callback: (message: ChatMessage) => void,
        onReactions?: (change: ReactionsChange) => void,
    ): () => void {
        return openStream(
            (signal) => this.client.listenForEvents({}, {signal, timeoutMs: NO_TIMEOUT}),
            (event) => {
                const message = messageOf(event)
                if (message) callback(message)
                const change = reactionsOf(event)
                if (change) onReactions?.(change)
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
        authorTag: message.authorTag,
        authorAdmin: message.authorAdmin,
        countryCode: message.countryId,
        text: message.text,
        reactions: message.reactions.map(decodedCount),
    }
}

export function reactionsOf(event: ChatEvent): ReactionsChange | undefined {
    if (event.event.case !== "reactions") return undefined

    return {
        messageId: event.event.value.messageId,
        reactions: event.event.value.reactions.map(decodedCount),
    }
}

function decodedCount(count: ReactionCountPb): ReactionCount {
    return {reaction: count.reaction, count: count.count, mine: count.mine}
}
