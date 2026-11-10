import {afterEach, describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {AntibotEvent as AntibotEventPb} from "../gen/grpc/antibot/v1/antibot_pb.ts"
import {ConnectAntibotBackend} from "./antibotBackend.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"

function holding(identity: string | undefined): SessionProvider & {
    token: ReturnType<typeof vi.fn>, identity: ReturnType<typeof vi.fn>, invalidate: ReturnType<typeof vi.fn>,
} {
    return {
        token: vi.fn(async () => "minted"),
        held: vi.fn(() => undefined),
        identity: vi.fn(async () => identity),
        heldIdentity: vi.fn(() => identity),
        invalidate: vi.fn(),
    }
}

function backendWith(methods: Record<string, unknown>, session: SessionProvider = holding("identity-1")) {
    return new ConnectAntibotBackend(methods as never, session)
}

const optionsOf = (call: ReturnType<typeof vi.fn>, n = 0) =>
    (call.mock.calls[n] as unknown[])[1] as {headers: Headers, timeoutMs?: number}

const failingWith = (error: ConnectError) => (): AsyncIterable<AntibotEventPb> => ({
    [Symbol.asyncIterator]: () => ({
        next: async () => {
            throw error
        },
    }),
})

afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
})

describe("ConnectAntibotBackend warnings", () => {
    it("opens the stream with the token held, never a mint, tells each warning and skips the heartbeats", async () => {
        const session = holding("identity-1")
        const listenForEvents = vi.fn(async function* () {
            yield new AntibotEventPb({event: {case: "heartbeat", value: {}}})
            yield new AntibotEventPb({event: {case: "warning", value: {}}})
            await new Promise(() => {})
        })
        const onWarned = vi.fn()

        const stop = backendWith({listenForEvents}, session).listenForWarnings(onWarned)
        await vi.waitFor(() => expect(onWarned).toHaveBeenCalledTimes(1))
        stop()

        expect(optionsOf(listenForEvents).headers.get(SESSION_HEADER)).toBe("identity-1")
        expect(optionsOf(listenForEvents).timeoutMs).toBe(0)
        expect(session.token).not.toHaveBeenCalled()
    })

    it("stops for good on a refused token", async () => {
        vi.useFakeTimers()
        const listenForEvents = vi.fn(failingWith(new ConnectError("no", Code.Unauthenticated)))
        const session = holding("expired")

        backendWith({listenForEvents}, session).listenForWarnings(() => {})
        await vi.advanceTimersByTimeAsync(60_000)

        expect(listenForEvents).toHaveBeenCalledTimes(1)
        expect(session.invalidate).not.toHaveBeenCalled()
    })

    it("reconnects after any other failure, and tells the warning the new stream sends", async () => {
        vi.useFakeTimers()
        vi.spyOn(console, "error").mockImplementation(() => {})
        const listenForEvents = vi.fn()
            .mockImplementationOnce(failingWith(new ConnectError("down", Code.Unavailable)))
            .mockImplementation(async function* () {
                yield new AntibotEventPb({event: {case: "warning", value: {}}})
                await new Promise(() => {})
            })
        const onWarned = vi.fn()

        const stop = backendWith({listenForEvents}).listenForWarnings(onWarned)
        await vi.advanceTimersByTimeAsync(1_000)
        stop()

        expect(listenForEvents).toHaveBeenCalledTimes(2)
        expect(onWarned).toHaveBeenCalledTimes(1)
    })
})

describe("ConnectAntibotBackend acknowledge", () => {
    it("acknowledges the warning with the identity token, never a mint", async () => {
        const session = holding("identity-1")
        const acknowledgeWarning = vi.fn(async () => ({}))

        await backendWith({acknowledgeWarning}, session).acknowledgeWarning()

        expect(acknowledgeWarning).toHaveBeenCalledWith({}, expect.anything())
        expect(optionsOf(acknowledgeWarning).headers.get(SESSION_HEADER)).toBe("identity-1")
        expect(session.token).not.toHaveBeenCalled()
    })

    it("reports a refusal as it is", async () => {
        const acknowledgeWarning = vi.fn(async () => {
            throw new ConnectError("no", Code.Unauthenticated)
        })

        const error = await backendWith({acknowledgeWarning}).acknowledgeWarning().catch((e) => e)

        expect(error).toBeInstanceOf(ConnectError)
        expect(acknowledgeWarning).toHaveBeenCalledTimes(1)
    })
})
