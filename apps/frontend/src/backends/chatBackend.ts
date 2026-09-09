import {
    ChatBlockedError,
    ChatHistoryGetter,
    ChatListener,
    ChatMessage,
    ChatRateLimitedError,
    ChatRejectedError,
    ChatSender,
    ChatUnavailableError,
    OutgoingMessage,
} from "./chat.ts";
import {ChatMessage as ChatMessagePb} from "../gen/grpc/chat/v1/chat_pb.ts";
import {ChatService} from "../gen/grpc/chat/v1/chat_connect.ts";
import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect";
import {createConnectTransport} from "@connectrpc/connect-web";
import {Config, openSocket, retrying, websocketUrl} from "./transport.ts";

const CHAT_ROUTE = "/ws/chat"

export function chatWebsocketUrl(baseUrl: string): string {
    return websocketUrl(baseUrl, CHAT_ROUTE)
}

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
        private config: Config,
        private client: PromiseClient<typeof ChatService>,
    ) {
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

    public listenForMessages(callback: (message: ChatMessage) => void): () => void {
        return openSocket(chatWebsocketUrl(this.config.baseUrl), (data) => {
            const message = decodeChatMessage(data)
            if (message) callback(message)
        })
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
        case Code.Unimplemented:
            return new ChatUnavailableError({cause: e})
        default:
            return e
    }
}

export function decodeChatMessage(data: unknown): ChatMessage | undefined {
    if (!(data instanceof ArrayBuffer)) {
        console.error("Ignoring a non-binary chat frame", data)
        return undefined
    }

    try {
        return decodedMessage(ChatMessagePb.fromBinary(new Uint8Array(data)))
    } catch (e) {
        console.error("Ignoring a malformed chat frame", e)
        return undefined
    }
}

function decodedMessage(message: ChatMessagePb): ChatMessage {
    return {
        id: message.id,
        sentAt: Number(message.sentAtUnixMs),
        authorName: message.authorName,
        authorTag: message.authorTag,
        countryCode: message.countryId,
        text: message.text,
    }
}
