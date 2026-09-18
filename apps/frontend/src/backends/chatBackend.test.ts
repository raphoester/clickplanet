import {afterEach, describe, expect, it, vi} from "vitest"
import {ChatServiceBackend, decodedMessage, messageOf, reactionsOf} from "./chatBackend.ts"
import {Code, ConnectError, PromiseClient} from "@connectrpc/connect"
import {
    ChatEvent,
    ChatMessage as ChatMessagePb,
    Heartbeat,
    Reaction,
    ReactionCount,
    ReactionsChanged,
} from "../gen/grpc/chat/v1/chat_pb.ts"
import {ChatService} from "../gen/grpc/chat/v1/chat_connect.ts"
import {
    ChatBlockedError,
    ChatMessageGoneError,
    ChatRateLimitedError,
    ChatRejectedError,
} from "./chat.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"

const outgoing = {authorName: "Ana", authorId: "author-1", countryCode: "fr", text: "hello", asAccount: false}

const session = (): SessionProvider => ({token: vi.fn(async () => "token-1"), held: vi.fn(() => "token-1"), invalidate: vi.fn()})

const headersOf = (call: unknown) =>
    ((call as ReturnType<typeof vi.fn>).mock.calls[0][1] as {headers: Headers}).headers

const proto = () => new ChatMessagePb({
    id: "message-1",
    sentAtUnixMs: BigInt(1_700_000_000_000),
    authorName: "Ana",
    authorTag: "4f2ca1",
    authorAdmin: true,
    countryId: "fr",
    text: "hello",
    reactions: [new ReactionCount({reaction: Reaction.CLOWN, count: 2, mine: true})],
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
            authorAdmin: true,
            countryCode: "fr",
            text: "hello",
            reactions: [{reaction: Reaction.CLOWN, count: 2, mine: true}],
        })
    })
})

describe("reactionsOf", () => {
    it("unwraps a reactions event, and nothing else", () => {
        const event = new ChatEvent({
            event: {
                case: "reactions",
                value: new ReactionsChanged({
                    messageId: "message-1",
                    reactions: [new ReactionCount({reaction: Reaction.SKULL, count: 3})],
                }),
            },
        })

        expect(reactionsOf(event)).toEqual({
            messageId: "message-1",
            reactions: [{reaction: Reaction.SKULL, count: 3, mine: false}],
        })
        expect(reactionsOf(new ChatEvent({event: {case: "message", value: proto()}}))).toBeUndefined()
    })
})

describe("ChatServiceBackend.react", () => {
    const reaction = {messageId: "message-1", reaction: Reaction.CLOWN, on: true, asAccount: true}

    it("sends the token for a player, none for a guest, and answers the counts", async () => {
        const react = vi.fn().mockResolvedValue({reactions: [new ReactionCount({reaction: Reaction.CLOWN, count: 1, mine: true})]})
        const client = {react} as unknown as PromiseClient<typeof ChatService>

        const counts = await new ChatServiceBackend(client, session()).react(reaction)
        await new ChatServiceBackend(client, session()).react({...reaction, asAccount: false})

        expect(counts).toEqual([{reaction: Reaction.CLOWN, count: 1, mine: true}])
        expect(react.mock.calls[0][0]).toEqual({messageId: "message-1", reaction: Reaction.CLOWN, on: true})
        expect((react.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("token-1")
        expect((react.mock.calls[1][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBeNull()
    })

    it("reads a message the server no longer shows as gone", async () => {
        const client = {
            react: () => Promise.reject(new ConnectError("gone", Code.NotFound)),
        } as unknown as PromiseClient<typeof ChatService>

        await expect(new ChatServiceBackend(client, session()).react(reaction)).rejects.toBeInstanceOf(ChatMessageGoneError)
    })
})

describe("messageOf", () => {
    it("unwraps a message event", () => {
        const event = new ChatEvent({event: {case: "message", value: proto()}})
        expect(messageOf(event)?.text).toBe("hello")
    })

    it("drops a heartbeat", () => {
        const heartbeat = new ChatEvent({event: {case: "heartbeat", value: new Heartbeat()}})
        expect(messageOf(heartbeat)).toBeUndefined()
    })

    it("drops an event case this build does not know", () => {
        // What a client sees when the backend adds a case: an unset oneof, not a crash.
        expect(messageOf(new ChatEvent())).toBeUndefined()
    })
})

describe("ChatServiceBackend.sendMessage", () => {
    it("returns the message the server stamped", async () => {
        const client = {
            sendMessage: vi.fn().mockResolvedValue({message: proto()}),
        } as unknown as PromiseClient<typeof ChatService>

        const sent = await new ChatServiceBackend(client, session()).sendMessage(outgoing)

        expect(sent.id).toBe("message-1")
        expect(sent.authorTag).toBe("4f2ca1")
        expect(client.sendMessage).toHaveBeenCalledWith({
            authorName: "Ana",
            authorId: "author-1",
            countryId: "fr",
            text: "hello",
        }, expect.anything())
    })

    // A guest who never clicked must not mint a session just to chat.
    it("sends a guest's message with no token, and asks for none", async () => {
        const sendMessage = vi.fn().mockResolvedValue({message: proto()})
        const guest = session()

        await new ChatServiceBackend({sendMessage} as unknown as PromiseClient<typeof ChatService>, guest)
            .sendMessage(outgoing)

        expect(guest.token).not.toHaveBeenCalled()
        expect(headersOf(sendMessage).has(SESSION_HEADER)).toBe(false)
    })

    it("sends a player's message with the click token", async () => {
        const sendMessage = vi.fn().mockResolvedValue({message: proto()})

        await new ChatServiceBackend({sendMessage} as unknown as PromiseClient<typeof ChatService>, session())
            .sendMessage({...outgoing, asAccount: true})

        expect(headersOf(sendMessage).get(SESSION_HEADER)).toBe("token-1")
    })

    it("sends as a guest when no token can be had", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const sendMessage = vi.fn().mockResolvedValue({message: proto()})
        const failing: SessionProvider = {
            token: vi.fn(async () => {
                throw new Error("no mint")
            }),
            held: vi.fn(() => undefined),
            invalidate: vi.fn(),
        }

        const sent = await new ChatServiceBackend({sendMessage} as unknown as PromiseClient<typeof ChatService>, failing)
            .sendMessage({...outgoing, asAccount: true})

        expect(sent.id).toBe("message-1")
        expect(headersOf(sendMessage).has(SESSION_HEADER)).toBe(false)
    })

    it("does not retry: a resent message would post twice", async () => {
        const sendMessage = vi.fn().mockRejectedValue(new ConnectError("down", Code.Unavailable))
        const client = {sendMessage} as unknown as PromiseClient<typeof ChatService>

        await expect(new ChatServiceBackend(client, session()).sendMessage(outgoing)).rejects.toThrow()
        expect(sendMessage).toHaveBeenCalledTimes(1)
    })

    const refusals = [
        {code: Code.ResourceExhausted, error: ChatRateLimitedError},
        {code: Code.PermissionDenied, error: ChatBlockedError},
        {code: Code.InvalidArgument, error: ChatRejectedError},
    ]

    it.each(refusals)("translates $code into its own error", async ({code, error}) => {
        const backend = new ChatServiceBackend(clientThatFails(new ConnectError("no", code)), session())

        await expect(backend.sendMessage(outgoing)).rejects.toBeInstanceOf(error)
    })

    it("leaves any other fault alone", async () => {
        const fault = new ConnectError("boom", Code.Internal)
        const backend = new ChatServiceBackend(clientThatFails(fault), session())

        await expect(backend.sendMessage(outgoing)).rejects.toBe(fault)
    })
})

describe("ChatServiceBackend.getHistory", () => {
    it("sends the token it holds, and never mints one", async () => {
        const getHistory = vi.fn().mockResolvedValue({messages: []})
        const client = {getHistory} as unknown as PromiseClient<typeof ChatService>
        const provider = session()

        await new ChatServiceBackend(client, provider).getHistory()

        expect((getHistory.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("token-1")
        expect(provider.token).not.toHaveBeenCalled()
    })

    it("decodes every message the server holds", async () => {
        const client = {
            getHistory: vi.fn().mockResolvedValue({messages: [proto(), proto()]}),
        } as unknown as PromiseClient<typeof ChatService>

        expect(await new ChatServiceBackend(client, session()).getHistory()).toHaveLength(2)
    })

    it("retries while the server cannot be reached", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const getHistory = vi.fn()
            .mockRejectedValueOnce(new ConnectError("down", Code.Unavailable))
            .mockResolvedValue({messages: [proto()]})
        const client = {getHistory} as unknown as PromiseClient<typeof ChatService>

        expect(await new ChatServiceBackend(client, session()).getHistory()).toHaveLength(1)
        expect(getHistory).toHaveBeenCalledTimes(2)
    })
})
