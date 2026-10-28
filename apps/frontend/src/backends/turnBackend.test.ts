import {describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {Timestamp} from "@bufbuild/protobuf"
import {Debrief as DebriefMessage, EntryKind, GamemasterEvent, Timeline, Turn as TurnMessage} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {BattleEnded, DayEnded, SeasonEnded} from "../gen/grpc/gamemaster/v1/events_pb.ts"
import {ConnectTurnBackend, turnOf} from "./turnBackend.ts"
import {Seasons} from "./season.ts"
import {BattleToCome, Debrief, Turn} from "./turn.ts"

const noWakeups = () => () => {}

const since = Date.UTC(2026, 9, 31, 21)
const endsAt = Date.UTC(2026, 9, 31, 23)
const at = (ms: number) => Timestamp.fromDate(new Date(ms))

function follow(listenForEvents: () => AsyncIterable<GamemasterEvent>) {
    const backend = new ConnectTurnBackend({listenForEvents: vi.fn(listenForEvents)} as never, noWakeups)
    const turns: Turn[] = []
    const battles: BattleToCome[][] = []
    const seasons: Seasons[] = []
    const debriefs: (Debrief | undefined)[] = []
    const stop = backend.follow({
        onTurn: (turn) => turns.push(turn),
        onBattles: (next) => battles.push(next),
        onSeasons: (next) => seasons.push(next),
        onDebrief: (next) => debriefs.push(next),
    })
    return {turns, battles, seasons, debriefs, stop}
}

describe("ConnectTurnBackend", () => {
    it("hands on each turn and each timeline the stream sends, and skips the heartbeats", async () => {
        const followed = follow(async function* () {
            yield new GamemasterEvent({event: {case: "turn", value: new TurnMessage({kind: EntryKind.PLAIN})}})
            yield new GamemasterEvent({event: {case: "heartbeat", value: {}}})
            yield new GamemasterEvent({event: {case: "timeline", value: new Timeline({battles: [{
                entryId: "finale", startsAt: at(since), endsAt: at(endsAt), description: "Triple points.", times: 3, endsSeason: true,
                roundId: "finale",
            }]})}})
            yield new GamemasterEvent({event: {case: "timeline", value: new Timeline()}})
            await new Promise(() => {})
        })
        await vi.waitFor(() => expect(followed.battles).toHaveLength(2))
        followed.stop()

        expect(followed.turns).toEqual([{kind: "plain", since: undefined, endsAt: undefined, description: "", entryId: "", roundId: "", times: 0}])
        expect(followed.battles).toEqual([
            [{entryId: "finale", startsAt: since, endsAt, description: "Triple points.", times: 3, endsSeason: true, roundId: "finale"}],
            [],
        ])
    })

    it("hands on the season that runs, the next one, and the day in progress the timeline names", async () => {
        const followed = follow(async function* () {
            yield new GamemasterEvent({event: {case: "timeline", value: new Timeline({
                current: {number: 0, startsAt: at(Date.UTC(2026, 9, 8)), endsAt: at(endsAt), bookId: "season-0"},
                next: {number: 1, startsAt: at(endsAt + 3_600_000), endsAt: at(Date.UTC(2026, 10, 30, 23)), bookId: "season-1"},
                today: {number: 21, startsAt: at(Date.UTC(2026, 9, 27, 21)), endsAt: at(Date.UTC(2026, 9, 28, 21)), roundId: "season-0/day-21"},
            })}})
            yield new GamemasterEvent({event: {case: "timeline", value: new Timeline()}})
            await new Promise(() => {})
        })
        await vi.waitFor(() => expect(followed.seasons).toHaveLength(2))
        followed.stop()

        expect(followed.seasons).toEqual([
            {
                current: {number: 0, startsAt: Date.UTC(2026, 9, 8), endsAt, bookId: "season-0"},
                next: {number: 1, startsAt: endsAt + 3_600_000, endsAt: Date.UTC(2026, 10, 30, 23), bookId: "season-1"},
                today: {number: 21, startsAt: Date.UTC(2026, 9, 27, 21), endsAt: Date.UTC(2026, 9, 28, 21), roundId: "season-0/day-21"},
            },
            {current: undefined, next: undefined, today: undefined},
        ])
    })

    it("reads the battle that plays, in milliseconds, as the wire carries it", () => {
        const sent = new TurnMessage({
            kind: EntryKind.BATTLE, since: at(since), endsAt: at(endsAt), description: "Triple points.",
            entryId: "0192f5a0-7c00-7000-8000-000000000031", roundId: "0192f5a0-7c00-7000-8000-000000000031", times: 3,
        })

        expect(turnOf(TurnMessage.fromBinary(sent.toBinary()))).toEqual({
            kind: "battle", since, endsAt, description: "Triple points.", entryId: "0192f5a0-7c00-7000-8000-000000000031",
            roundId: "0192f5a0-7c00-7000-8000-000000000031", times: 3,
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
            const onDebrief = vi.fn()

            backend.follow({onTurn, onBattles, onSeasons, onDebrief})
            await vi.advanceTimersByTimeAsync(60_000)

            expect(onTurn.mock.calls).toEqual([[{kind: "plain", description: "", entryId: "", roundId: "", times: 0}]])
            expect(onBattles.mock.calls).toEqual([[[]]])
            expect(onSeasons.mock.calls).toEqual([[{}]])
            expect(onDebrief.mock.calls).toEqual([[undefined]])
            expect(listenForEvents).toHaveBeenCalledTimes(1)
        }
        vi.useRealTimers()
    })

    it("hands on each debrief, in the gamemaster's words and with the ids to read its numbers by", async () => {
        const debrief = (moment: DebriefMessage["moment"]) =>
            new GamemasterEvent({event: {case: "debrief", value: new DebriefMessage({moment})}})
        const followed = follow(async function* () {
            yield debrief({case: "day", value: new DayEnded({season: 0, day: 24, roundId: "season-0/day-24", endedAt: at(since)})})
            yield debrief({case: "battle", value: new BattleEnded({
                season: 0, number: 2, entryId: "finale", times: 3, endsSeason: true, roundId: "finale", endedAt: at(endsAt),
            })})
            yield debrief({case: "season", value: new SeasonEnded({
                season: 0, bookId: "season-0", endedAt: at(endsAt), finaleRoundId: "finale", finaleBattle: new BattleEnded({
                    season: 0, number: 2, entryId: "finale", times: 3, endsSeason: true, roundId: "finale", endedAt: at(endsAt),
                }),
            })})
            yield debrief({case: "season", value: new SeasonEnded({season: 0, bookId: "season-0", endedAt: at(endsAt)})})
            yield debrief({case: undefined})
            await new Promise(() => {})
        })
        await vi.waitFor(() => expect(followed.debriefs).toHaveLength(5))
        followed.stop()

        expect(followed.debriefs).toEqual([
            {kind: "day", season: 0, number: 24, roundId: "season-0/day-24", endedAt: since},
            {kind: "battle", season: 0, number: 2, entryId: "finale", times: 3, endsSeason: true, roundId: "finale", endedAt: endsAt},
            {
                kind: "season", season: 0, bookId: "season-0", endedAt: endsAt, finaleRoundId: "finale",
                finaleBattle: {kind: "battle", season: 0, number: 2, entryId: "finale", times: 3, endsSeason: true, roundId: "finale", endedAt: endsAt},
            },
            {kind: "season", season: 0, bookId: "season-0", endedAt: endsAt, finaleRoundId: "", finaleBattle: undefined},
            undefined,
        ])
    })

    it("refuses a turn with no kind, or a kind this build does not know", () => {
        expect(() => turnOf(new TurnMessage({kind: EntryKind.UNSPECIFIED}))).toThrow("no kind")
        expect(() => turnOf(new TurnMessage({kind: 7 as EntryKind}))).toThrow("does not know")
    })
})
