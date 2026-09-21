import {afterEach, describe, expect, it, vi} from "vitest"
import {announcementOf, ChatServiceBackend, decodedAnnouncement, decodedMessage, messageOf, reactionsOf} from "./chatBackend.ts"
import {Code, ConnectError, PromiseClient} from "@connectrpc/connect"
import {
    Announcement,
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
    ChatNoSessionError,
    ChatRateLimitedError,
    ChatRejectedError,
} from "./chat.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"

const outgoing = {authorId: "author-1", countryCode: "fr", text: "hello"}

const session = (): SessionProvider => ({token: vi.fn(async () => "token-1"), held: vi.fn(() => "token-1"), invalidate: vi.fn()})

/** Nothing held, as before a first click: `token` mints. */
const unheld = (): SessionProvider => ({token: vi.fn(async () => "minted"), held: vi.fn(() => undefined), invalidate: vi.fn()})

const noMint = (): SessionProvider => ({
    token: vi.fn(async () => {
        throw new Error("no mint")
    }),
    held: vi.fn(() => undefined),
    invalidate: vi.fn(),
})

const refusedOnce = () => vi.fn()
    .mockRejectedValueOnce(new ConnectError("no token", Code.Unauthenticated))

const headersOf = (call: unknown) =>
    ((call as ReturnType<typeof vi.fn>).mock.calls[0][1] as {headers: Headers}).headers

const proto = () => new ChatMessagePb({
    id: "message-1",
    sentAtUnixMs: BigInt(1_700_000_000_000),
    authorName: "Ana",
    authorAdmin: true,
    countryId: "fr",
    text: "hello",
    reactions: [new ReactionCount({reaction: Reaction.CLOWN, count: 2, mine: true, reactors: ["Ana", "Bo"]})],
    reactionsVersion: BigInt(3),
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
            authorAdmin: true,
            countryCode: "fr",
            text: "hello",
            reactions: [{reaction: Reaction.CLOWN, count: 2, mine: true, reactors: ["Ana", "Bo"]}],
            reactionsVersion: 3,
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
                    reactions: [new ReactionCount({reaction: Reaction.SKULL, count: 3, reactors: ["Bo"]})],
                    version: BigInt(8),
                }),
            },
        })

        expect(reactionsOf(event)).toEqual({
            messageId: "message-1",
            reactions: [{reaction: Reaction.SKULL, count: 3, mine: false, reactors: ["Bo"]}],
            version: 8,
        })
        expect(reactionsOf(new ChatEvent({event: {case: "message", value: proto()}}))).toBeUndefined()
    })
})

describe("ChatServiceBackend.react", () => {
    const reaction = {messageId: "message-1", reaction: Reaction.CLOWN, on: true}
    const answer = {
        reactions: [new ReactionCount({reaction: Reaction.CLOWN, count: 1, mine: true, reactors: ["Ana"]})],
        version: BigInt(2),
    }

    it("sends the click token and answers the counts", async () => {
        const react = vi.fn().mockResolvedValue(answer)
        const client = {react} as unknown as PromiseClient<typeof ChatService>

        const counts = await new ChatServiceBackend(client, session()).react(reaction)

        expect(counts).toEqual({
            messageId: "message-1",
            reactions: [{reaction: Reaction.CLOWN, count: 1, mine: true, reactors: ["Ana"]}],
            version: 2,
        })
        expect(react.mock.calls[0][0]).toEqual({messageId: "message-1", reaction: Reaction.CLOWN, on: true})
        expect(headersOf(react).get(SESSION_HEADER)).toBe("token-1")
    })

    // Every caller reacts as its account, guests included.
    it("mints a token when none is held", async () => {
        const react = vi.fn().mockResolvedValue(answer)
        const guest = unheld()

        await new ChatServiceBackend({react} as unknown as PromiseClient<typeof ChatService>, guest).react(reaction)

        expect(guest.token).toHaveBeenCalled()
        expect(headersOf(react).get(SESSION_HEADER)).toBe("minted")
    })

    it("fails when no token can be had, and sends nothing", async () => {
        const react = vi.fn().mockResolvedValue(answer)

        await expect(new ChatServiceBackend({react} as unknown as PromiseClient<typeof ChatService>, noMint()).react(reaction))
            .rejects.toBeInstanceOf(ChatNoSessionError)
        expect(react).not.toHaveBeenCalled()
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

describe("decodedAnnouncement", () => {
    const bomb = (payload: string, kind = "bomb") => new Announcement({
        id: "announcement-1",
        announcedAtUnixMs: BigInt(1_700_000_000_000),
        kind,
        payload,
    })

    it("reads a bomb on land", () => {
        expect(decodedAnnouncement(bomb(`{"country":"fr","ground":"de","tile":42,"cleared":3}`))).toEqual({
            kind: "bomb",
            id: "announcement-1",
            announcedAt: 1_700_000_000_000,
            country: "fr",
            ground: "de",
            tile: 42,
            cleared: 3,
        })
    })

    it("reads a bomb in the sea, with no ground and no tile", () => {
        const announcement = decodedAnnouncement(bomb(`{"country":"fr","cleared":0}`))

        expect(announcement?.tile).toBeUndefined()
        expect(announcement?.ground).toBeUndefined()
        expect(announcement?.cleared).toBe(0)
    })

    it("drops a kind it does not know, and a payload that is not the kind's", () => {
        expect(decodedAnnouncement(bomb(`{"country":"fr"}`, "meteor"))).toBeUndefined()
        expect(decodedAnnouncement(bomb(`not json`))).toBeUndefined()
        expect(decodedAnnouncement(bomb(`{"cleared":3}`))).toBeUndefined()
        expect(decodedAnnouncement(bomb(`null`))).toBeUndefined()
    })

    it("comes off the stream as the announcement case only", () => {
        const event = new ChatEvent({event: {case: "announcement", value: bomb(`{"country":"fr","cleared":0}`)}})

        expect(announcementOf(event)?.country).toBe("fr")
        expect(announcementOf(new ChatEvent({event: {case: "message", value: proto()}}))).toBeUndefined()
    })
})

describe("ChatServiceBackend.sendMessage", () => {
    it("returns the message the server stamped", async () => {
        const client = {
            sendMessage: vi.fn().mockResolvedValue({message: proto()}),
        } as unknown as PromiseClient<typeof ChatService>

        const sent = await new ChatServiceBackend(client, session()).sendMessage(outgoing)

        expect(sent.id).toBe("message-1")
        expect(sent.authorName).toBe("Ana")
        expect(client.sendMessage).toHaveBeenCalledWith({
            authorId: "author-1",
            countryId: "fr",
            text: "hello",
        }, expect.anything())
    })

    it("sends the click token it holds", async () => {
        const sendMessage = vi.fn().mockResolvedValue({message: proto()})

        await new ChatServiceBackend({sendMessage} as unknown as PromiseClient<typeof ChatService>, session())
            .sendMessage(outgoing)

        expect(headersOf(sendMessage).get(SESSION_HEADER)).toBe("token-1")
    })

    // The server names the sender by its account, a guest's included.
    it("mints a token when none is held, as a click does", async () => {
        const sendMessage = vi.fn().mockResolvedValue({message: proto()})
        const guest = unheld()

        await new ChatServiceBackend({sendMessage} as unknown as PromiseClient<typeof ChatService>, guest)
            .sendMessage(outgoing)

        expect(guest.token).toHaveBeenCalled()
        expect(headersOf(sendMessage).get(SESSION_HEADER)).toBe("minted")
    })

    it("fails when no token can be had, and sends nothing", async () => {
        const sendMessage = vi.fn().mockResolvedValue({message: proto()})

        await expect(new ChatServiceBackend({sendMessage} as unknown as PromiseClient<typeof ChatService>, noMint())
            .sendMessage(outgoing)).rejects.toBeInstanceOf(ChatNoSessionError)
        expect(sendMessage).not.toHaveBeenCalled()
    })

    it("tries once more with a fresh token when the server refuses the one it sent", async () => {
        const sendMessage = refusedOnce().mockResolvedValue({message: proto()})
        const provider = session()

        const sent = await new ChatServiceBackend({sendMessage} as unknown as PromiseClient<typeof ChatService>, provider)
            .sendMessage(outgoing)

        expect(sent.id).toBe("message-1")
        expect(provider.invalidate).toHaveBeenCalledTimes(1)
        expect(sendMessage).toHaveBeenCalledTimes(2)
    })

    it("reads a second refusal of the token as no session", async () => {
        const sendMessage = vi.fn().mockRejectedValue(new ConnectError("no token", Code.Unauthenticated))

        await expect(new ChatServiceBackend({sendMessage} as unknown as PromiseClient<typeof ChatService>, session())
            .sendMessage(outgoing)).rejects.toBeInstanceOf(ChatNoSessionError)
        expect(sendMessage).toHaveBeenCalledTimes(2)
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
        const getHistory = vi.fn().mockResolvedValue({messages: [], announcements: []})
        const client = {getHistory} as unknown as PromiseClient<typeof ChatService>
        const provider = session()

        await new ChatServiceBackend(client, provider).getHistory()

        expect((getHistory.mock.calls[0][1] as {headers: Headers}).headers.get(SESSION_HEADER)).toBe("token-1")
        expect(provider.token).not.toHaveBeenCalled()
    })

    it("decodes every message the server holds", async () => {
        const client = {
            getHistory: vi.fn().mockResolvedValue({messages: [proto(), proto()], announcements: []}),
        } as unknown as PromiseClient<typeof ChatService>

        expect((await new ChatServiceBackend(client, session()).getHistory()).messages).toHaveLength(2)
    })

    it("retries while the server cannot be reached", async () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const getHistory = vi.fn()
            .mockRejectedValueOnce(new ConnectError("down", Code.Unavailable))
            .mockResolvedValue({messages: [proto()], announcements: []})
        const client = {getHistory} as unknown as PromiseClient<typeof ChatService>

        expect((await new ChatServiceBackend(client, session()).getHistory()).messages).toHaveLength(1)
        expect(getHistory).toHaveBeenCalledTimes(2)
    })
})
