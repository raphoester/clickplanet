import {
    ChatBlockedError,
    ChatFeedEvent,
    ChatHistoryGetter,
    ChatListener,
    ChatMessage,
    ChatRateLimitedError,
    ChatRejectedError,
    ChatSender,
    OutgoingMessage,
} from "./chat.ts";
import {ChatEvent as ChatEventPb, ChatMessage as ChatMessagePb} from "../gen/grpc/chat/v1/chat_pb.ts";
import {ChatService} from "../gen/grpc/chat/v1/chat_connect.ts";
import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect";
import {createConnectTransport} from "@connectrpc/connect-web";
import {Config, NO_TIMEOUT, openStream, retrying} from "./transport.ts";

export function newChatServiceClient(config: Config): PromiseClient<typeof ChatService> {
    return createPromiseClient(ChatService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class ChatServiceBackend implements ChatSender, ChatHistoryGetter, ChatListener {
    constructor(private client: PromiseClient<typeof ChatService>) {
    }

    public async sendMessage(message: OutgoingMessage): Promise<ChatMessage> {
        try {
            const res = await this.client.sendMessage({
                authorName: message.authorName,
                authorId: message.authorId,
                countryId: message.countryCode,
                text: message.text,
            })

            if (!res.message) throw new ChatRejectedError()

            return decodedMessage(res.message)
        } catch (e) {
            throw translate(e)
        }
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

    public listenForEvents(callback: (event: ChatFeedEvent) => void): () => void {
        return openStream(
            (signal) => this.client.listenForEvents({}, {signal, timeoutMs: NO_TIMEOUT}),
            (event) => {
                const decoded = feedEventOf(event)
                if (decoded) callback(decoded)
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
 * Heartbeats are dropped, as is an event case this build does not know, which
 * reads as an unset `oneof`.
 */
export function feedEventOf(event: ChatEventPb): ChatFeedEvent | undefined {
    switch (event.event.case) {
        case "message":
            return {kind: "message", message: decodedMessage(event.event.value)}
        case "memberRedacted":
            return {kind: "redacted", authorTag: event.event.value.authorTag}
        default:
            return undefined
    }
}

export function decodedMessage(message: ChatMessagePb): ChatMessage {
    return {
        id: message.id,
        sentAt: Number(message.sentAtUnixMs),
        authorName: message.authorName,
        authorTag: message.authorTag,
        countryCode: message.countryId,
        text: message.text,
        redacted: message.redacted,
    }
}
