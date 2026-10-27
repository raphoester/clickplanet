import {describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {Timestamp} from "@bufbuild/protobuf"
import {EntryKind, GamemasterEvent, Timeline, Turn as TurnMessage} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {ConnectTurnBackend, turnOf} from "./turnBackend.ts"
import {Seasons} from "./season.ts"
import {BattleToCome, Turn} from "./turn.ts"

const noWakeups = () => () => {}

const since = Date.UTC(2026, 9, 31, 21)
const endsAt = Date.UTC(2026, 9, 31, 23)
const at = (ms: number) => Timestamp.fromDate(new Date(ms))

function follow(listenForEvents: () => AsyncIterable<GamemasterEvent>) {
    const backend = new ConnectTurnBackend({listenForEvents: vi.fn(listenForEvents)} as never, noWakeups)
    const turns: Turn[] = []
    const battles: BattleToCome[][] = []
    const seasons: Seasons[] = []
    const stop = backend.follow({
        onTurn: (turn) => turns.push(turn), onBattles: (next) => battles.push(next), onSeasons: (next) => seasons.push(next),
    })
    return {turns, battles, seasons, stop}
}

describe("ConnectTurnBackend", () => {
    it("hands on each turn and each timeline the stream sends, and skips the heartbeats", async () => {
        const followed = follow(async function* () {
            yield new GamemasterEvent({event: {case: "turn", value: new TurnMessage({kind: EntryKind.PLAIN})}})
            yield new GamemasterEvent({event: {case: "heartbeat", value: {}}})
            yield new GamemasterEvent({event: {case: "timeline", value: new Timeline({battles: [{
                entryId: "finale", startsAt: at(since), endsAt: at(endsAt), description: "Triple points.",
            }]})}})
            yield new GamemasterEvent({event: {case: "timeline", value: new Timeline()}})
            await new Promise(() => {})
        })
        await vi.waitFor(() => expect(followed.battles).toHaveLength(2))
        followed.stop()

        expect(followed.turns).toEqual([{kind: "plain", since: undefined, endsAt: undefined, description: "", entryId: ""}])
        expect(followed.battles).toEqual([
            [{entryId: "finale", startsAt: since, endsAt, description: "Triple points."}],
            [],
        ])
    })

    it("hands on the season that runs and the next one the timeline names", async () => {
        const followed = follow(async function* () {
            yield new GamemasterEvent({event: {case: "timeline", value: new Timeline({
                current: {number: 0, startsAt: at(Date.UTC(2026, 9, 3)), endsAt: at(endsAt)},
                next: {number: 1, startsAt: at(endsAt + 3_600_000), endsAt: at(Date.UTC(2026, 10, 30, 23))},
            })}})
            yield new GamemasterEvent({event: {case: "timeline", value: new Timeline()}})
            await new Promise(() => {})
        })
        await vi.waitFor(() => expect(followed.seasons).toHaveLength(2))
        followed.stop()

        expect(followed.seasons).toEqual([
            {
                current: {number: 0, startsAt: Date.UTC(2026, 9, 3), endsAt},
                next: {number: 1, startsAt: endsAt + 3_600_000, endsAt: Date.UTC(2026, 10, 30, 23)},
            },
            {current: undefined, next: undefined},
        ])
    })

    it("reads the battle that plays, in milliseconds, as the wire carries it", () => {
        const sent = new TurnMessage({
            kind: EntryKind.BATTLE, since: at(since), endsAt: at(endsAt), description: "Triple points.",
            entryId: "0192f5a0-7c00-7000-8000-000000000031",
        })

        expect(turnOf(TurnMessage.fromBinary(sent.toBinary()))).toEqual({
            kind: "battle", since, endsAt, description: "Triple points.", entryId: "0192f5a0-7c00-7000-8000-000000000031",
        })
    })

    it("plays plain with no battle to come on a server without a schedule, and does not reconnect", async () => {
        vi.useFakeTimers()
        for (const code of [Code.Unimplemented, Code.NotFound]) {
            const listenForEvents = vi.fn((): AsyncIterable<GamemasterEvent> => ({
                [Symbol.asyncIterator]: () => ({
                    next: async () => {
                        throw new ConnectError("not here", code)
                    },
                }),
            }))
            const backend = new ConnectTurnBackend({listenForEvents} as never, noWakeups)
            const onTurn = vi.fn()
            const onBattles = vi.fn()
            const onSeasons = vi.fn()

            backend.follow({onTurn, onBattles, onSeasons})
            await vi.advanceTimersByTimeAsync(60_000)

            expect(onTurn.mock.calls).toEqual([[{kind: "plain", description: "", entryId: ""}]])
            expect(onBattles.mock.calls).toEqual([[[]]])
            expect(onSeasons.mock.calls).toEqual([[{}]])
            expect(listenForEvents).toHaveBeenCalledTimes(1)
        }
        vi.useRealTimers()
    })

    it("refuses a turn with no kind, or a kind this build does not know", () => {
        expect(() => turnOf(new TurnMessage({kind: EntryKind.UNSPECIFIED}))).toThrow("no kind")
        expect(() => turnOf(new TurnMessage({kind: 7 as EntryKind}))).toThrow("does not know")
    })
})
