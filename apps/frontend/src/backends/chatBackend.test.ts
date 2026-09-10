import {afterEach, describe, expect, it, vi} from "vitest"
import {ChatServiceBackend, decodedMessage} from "./chatBackend.ts"
import {Code, ConnectError, PromiseClient} from "@connectrpc/connect"
import {ChatMessage as ChatMessagePb} from "../gen/grpc/chat/v1/chat_pb.ts"
import {ChatService} from "../gen/grpc/chat/v1/chat_connect.ts"
import {
    ChatBlockedError,
    ChatRateLimitedError,
    ChatRejectedError,
    ChatUnavailableError,
} from "./chat.ts"

const outgoing = {authorName: "Ana", authorId: "author-1", countryCode: "fr", text: "hello"}

const proto = () => new ChatMessagePb({
    id: "message-1",
    sentAtUnixMs: BigInt(1_700_000_000_000),
    authorName: "Ana",
    authorTag: "4f2ca1",
    countryId: "fr",
    text: "hello",
})

function clientThatFails(error: unknown): PromiseClient<typeof ChatService> {
    return {
        sendMessage: () => Promise.reject(error),
        getHistory: () => Promise.reject(error),
    } as unknown as PromiseClient<typeof ChatService>
}

afterEach(() => vi.restoreAllMocks())

describe("decodedMessage", () => {
    it("maps a broadcast message onto the shape the chat consumes", () => {
        expect(decodedMessage(proto())).toEqual({
            id: "message-1",
            sentAt: 1_700_000_000_000,
            authorName: "Ana",
            authorTag: "4f2ca1",
            countryCode: "fr",
            text: "hello",
        })
    })
})

describe("ChatServiceBackend.sendMessage", () => {
    it("returns the message the server stamped", async () => {
        const client = {
            sendMessage: vi.fn().mockResolvedValue({message: proto()}),
        } as unknown as PromiseClient<typeof ChatService>

        const sent = await new ChatServiceBackend(client).sendMessage(outgoing)

        expect(sent.id).toBe("message-1")
        expect(sent.authorTag).toBe("4f2ca1")
        expect(client.sendMessage).toHaveBeenCalledWith({
            authorName: "Ana",
            authorId: "author-1",
            countryId: "fr",
            text: "hello",
        })
    })

    it("does not retry: a resent message would post twice", async () => {
        const sendMessage = vi.fn().mockRejectedValue(new ConnectError("down", Code.Unavailable))
        const client = {sendMessage} as unknown as PromiseClient<typeof ChatService>

        await expect(new ChatServiceBackend(client).sendMessage(outgoing)).rejects.toThrow()
        expect(sendMessage).toHaveBeenCalledTimes(1)
    })

    const refusals = [
        {code: Code.ResourceExhausted, error: ChatRateLimitedError},
        {code: Code.PermissionDenied, error: ChatBlockedError},
        {code: Code.InvalidArgument, error: ChatRejectedError},
        {code: Code.Unimplemented, error: ChatUnavailableError},
    ]

    it.each(refusals)("translates $code into its own error", async ({code, error}) => {
        const backend = new ChatServiceBackend(clientThatFails(new ConnectError("no", code)))

        await expect(backend.sendMessage(outgoing)).rejects.toBeInstanceOf(error)
    })

    it("leaves any other fault alone", async () => {
        const fault = new ConnectError("boom", Code.Internal)
        const backend = new ChatServiceBackend(clientThatFails(fault))

        await expect(backend.sendMessage(outgoing)).rejects.toBe(fault)
    })
})

describe("ChatServiceBackend.getHistory", () => {
    it("decodes every message the server holds", async () => {
        const client = {
            getHistory: vi.fn().mockResolvedValue({messages: [proto(), proto()]}),
        } as unknown as PromiseClient<typeof ChatService>

        expect(await new ChatServiceBackend(client).getHistory()).toHaveLength(2)
    })

    it("retries while the server cannot be reached", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const getHistory = vi.fn()
            .mockRejectedValueOnce(new ConnectError("down", Code.Unavailable))
            .mockResolvedValue({messages: [proto()]})
        const client = {getHistory} as unknown as PromiseClient<typeof ChatService>

        expect(await new ChatServiceBackend(client).getHistory()).toHaveLength(1)
        expect(getHistory).toHaveBeenCalledTimes(2)
    })

    it("reports a server with chat switched off as unavailable", async () => {
        const backend = new ChatServiceBackend(clientThatFails(new ConnectError("no such handler", Code.Unimplemented)),
        )

        await expect(backend.getHistory()).rejects.toBeInstanceOf(ChatUnavailableError)
    })
})
