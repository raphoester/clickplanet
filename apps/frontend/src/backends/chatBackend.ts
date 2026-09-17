import {
    ChatBlockedError,
    ChatHistoryGetter,
    ChatListener,
    ChatMessage,
    ChatRateLimitedError,
    ChatRejectedError,
    ChatSender,
    OutgoingMessage,
} from "./chat.ts";
import {ChatEvent, ChatMessage as ChatMessagePb} from "../gen/grpc/chat/v1/chat_pb.ts";
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

export class ChatServiceBackend implements ChatSender, ChatHistoryGetter, ChatListener {
    constructor(
        private client: PromiseClient<typeof ChatService>,
        private readonly session: SessionProvider,
    ) {
    }

    public async sendMessage(message: OutgoingMessage): Promise<ChatMessage> {
        const headers = await this.headersFor(message)

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
    private async headersFor(message: OutgoingMessage): Promise<Headers> {
        const headers = new Headers()
        if (!message.asAccount) return headers

        const token = await this.session.token().catch((e) => {
            console.error("No session for the chat: sending as a guest", e)
            return undefined
        })
        if (token) headers.set(SESSION_HEADER, token)
        return headers
    }

    public async getHistory(signal?: AbortSignal): Promise<ChatMessage[]> {
        try {
            const res = await retrying(
                () => this.client.getHistory({}, {signal}),
                "chat history",
                signal,
            )

            return res.messages.map(decodedMessage)
        } catch (e) {
            throw translate(e)
        }
    }

    public listenForMessages(callback: (message: ChatMessage) => void): () => void {
        return openStream(
            (signal) => this.client.listenForEvents({}, {signal, timeoutMs: NO_TIMEOUT}),
            (event) => {
                const message = messageOf(event)
                if (message) callback(message)
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
        countryCode: message.countryId,
        text: message.text,
    }
}
