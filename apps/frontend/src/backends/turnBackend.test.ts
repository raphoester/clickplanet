import {describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {Timestamp} from "@bufbuild/protobuf"
import {EntryKind, GetTurnResponse} from "../gen/grpc/metagame/v1/metagame_pb.ts"
import {ConnectTurnBackend, turnOf} from "./turnBackend.ts"

const noWakeups = () => () => {}

const backendWith = (getTurn: () => Promise<GetTurnResponse>) =>
    new ConnectTurnBackend({getTurn: vi.fn(getTurn)} as never, noWakeups)

const since = Date.UTC(2026, 9, 31, 21)
const endsAt = Date.UTC(2026, 9, 31, 23)

describe("ConnectTurnBackend", () => {
    it("reads the battle that plays, in milliseconds", async () => {
        const backend = backendWith(async () => new GetTurnResponse({turn: {
            kind: EntryKind.BATTLE,
            since: Timestamp.fromDate(new Date(since)),
            endsAt: Timestamp.fromDate(new Date(endsAt)),
            description: "Triple points.",
        }}))

        expect(await backend.turn()).toEqual({kind: "battle", since, endsAt, description: "Triple points."})
    })

    it("reads the battle a server sends in binary, as the wire carries it", async () => {
        const sent = new GetTurnResponse({turn: {kind: EntryKind.BATTLE, description: "Triple points."}})
        const backend = backendWith(async () => GetTurnResponse.fromBinary(sent.toBinary()))

        expect((await backend.turn()).kind).toBe("battle")
    })

    it("reads a plain turn with no entry", async () => {
        expect(await backendWith(async () => new GetTurnResponse({turn: {kind: EntryKind.PLAIN}})).turn())
            .toEqual({kind: "plain", since: undefined, endsAt: undefined, description: ""})
    })

    it("plays plain on a server without a schedule", async () => {
        const backend = backendWith(async () => {
            throw new ConnectError("not here", Code.Unimplemented)
        })

        expect(await backend.turn()).toEqual({kind: "plain", description: ""})
    })

    it("passes on a refusal it does not know", async () => {
        const backend = backendWith(async () => {
            throw new ConnectError("no", Code.PermissionDenied)
        })

        await expect(backend.turn()).rejects.toThrow(ConnectError)
    })

    it("refuses a turn with no kind, or a kind this build does not know", () => {
        expect(() => turnOf(new GetTurnResponse({turn: {kind: EntryKind.UNSPECIFIED}}))).toThrow("no kind")
        expect(() => turnOf(new GetTurnResponse({turn: {kind: 7 as EntryKind}}))).toThrow("does not know")
    })

    it("tells of a change when the page wakes", () => {
        const wake = vi.fn()
        let woken = () => {}
        const backend = new ConnectTurnBackend({} as never, (onWake) => {
            woken = onWake
            return () => {}
        })

        backend.changes(wake)
        woken()

        expect(wake).toHaveBeenCalledTimes(1)
    })
})
