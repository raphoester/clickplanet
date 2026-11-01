import {Timestamp} from "@bufbuild/protobuf"
import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {GamemasterService} from "../gen/grpc/gamemaster/v1/gamemaster_connect.ts"
import {
    Day as DayMessage,
    Debrief as DebriefMessage,
    EntryKind,
    GamemasterEvent,
    Season as SeasonMessage,
    Timeline,
    Turn as TurnMessage,
} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {BattleEnded as BattleEndedMessage} from "../gen/grpc/gamemaster/v1/events_pb.ts"
import {LedgerTotals} from "../gen/grpc/planet/v1/totals_pb.ts"
import {Day, NO_SEASONS, Season, Seasons, SeasonTotals} from "./season.ts"
import {Config, NO_TIMEOUT, openStream, pageWakeups, Wakeups} from "./transport.ts"
import {BattleEnded, BattleToCome, Debrief, PLAIN_TURN, Turn, TurnBackend, TurnKind, TurnListener} from "./turn.ts"

export function newGamemasterServiceClient(config: Config): PromiseClient<typeof GamemasterService> {
    return createPromiseClient(GamemasterService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

// A season's first read scans planet's ledger for seconds.
const TOTALS_TIMEOUT_MS = 60_000

export class ConnectTurnBackend implements TurnBackend {
    constructor(
        private readonly client: PromiseClient<typeof GamemasterService>,
        private readonly wakeups: Wakeups = pageWakeups,
    ) {
    }

    public follow({onTurn, onBattles, onSeasons, onDebrief}: TurnListener): () => void {
        const client = this.client
        let stop = () => {}
        stop = openStream(
            async function* (signal) {
                try {
                    yield* client.listenForEvents({}, {signal, timeoutMs: NO_TIMEOUT})
                } catch (e) {
                    if (e instanceof ConnectError && (e.code === Code.Unimplemented || e.code === Code.NotFound)) {
                        onTurn(PLAIN_TURN)
                        onBattles([])
                        onSeasons(NO_SEASONS)
                        onDebrief(undefined)
                        stop()
                        return
                    }
                    throw e
                }
            },
            (event: GamemasterEvent) => {
                switch (event.event.case) {
                    case "turn":
                        onTurn(turnOf(event.event.value))
                        return
                    case "timeline":
                        onBattles(battlesOf(event.event.value))
                        onSeasons(seasonsOf(event.event.value))
                        return
                    case "debrief":
                        onDebrief(debriefOf(event.event.value))
                        return
                    case "heartbeat":
                    case undefined:
                        return
                }
            },
            "gamemaster",
            {wakeups: this.wakeups},
        )
        return () => stop()
    }

    public async seasonTotals(season: number): Promise<SeasonTotals | undefined> {
        try {
            const res = await this.client.getSeasonTotals({season}, {timeoutMs: TOTALS_TIMEOUT_MS})
            return res.totals && totalsOf(res.totals)
        } catch (e) {
            if (e instanceof ConnectError && (e.code === Code.Unimplemented || e.code === Code.NotFound)) return undefined
            throw e
        }
    }
}

function totalsOf(totals: LedgerTotals): SeasonTotals {
    return {
        takes: Number(totals.takes),
        strikes: Number(totals.strikes),
        spreads: Number(totals.spreads),
        encloses: Number(totals.encloses),
        bombs: Number(totals.bombs),
        shields: Number(totals.shields),
        fortifies: Number(totals.fortifies),
        players: Number(totals.accounts),
    }
}

export function turnOf(turn: TurnMessage): Turn {
    return {
        kind: kindOf(turn.kind),
        since: turn.since?.toDate().getTime(),
        endsAt: turn.endsAt?.toDate().getTime(),
        description: turn.description,
        entryId: turn.entryId,
        roundId: turn.roundId,
        times: turn.times,
        endsSeason: turn.endsSeason,
    }
}

export function battlesOf(timeline: Timeline): BattleToCome[] {
    return timeline.battles.flatMap((battle) => battle.startsAt && battle.endsAt
        ? [{
            entryId: battle.entryId,
            startsAt: battle.startsAt.toDate().getTime(),
            endsAt: battle.endsAt.toDate().getTime(),
            description: battle.description,
            times: battle.times,
            endsSeason: battle.endsSeason,
            roundId: battle.roundId,
        }]
        : [])
}

export function seasonsOf(timeline: Timeline): Seasons {
    return {
        current: seasonOf(timeline.current),
        next: seasonOf(timeline.next),
        today: dayOf(timeline.today),
        ended: timeline.ended.flatMap((season) => seasonOf(season) ?? []),
    }
}

function seasonOf(season: SeasonMessage | undefined): Season | undefined {
    if (!season?.startsAt || !season.endsAt) return undefined
    return {
        number: season.number,
        startsAt: season.startsAt.toDate().getTime(),
        endsAt: season.endsAt.toDate().getTime(),
        bookId: season.bookId,
        finaleRoundId: season.finaleRoundId || undefined,
    }
}

function dayOf(day: DayMessage | undefined): Day | undefined {
    if (!day?.startsAt || !day.endsAt) return undefined
    return {number: day.number, startsAt: day.startsAt.toDate().getTime(), endsAt: day.endsAt.toDate().getTime(), roundId: day.roundId}
}

export function debriefOf(debrief: DebriefMessage): Debrief | undefined {
    const moment = debrief.moment
    switch (moment.case) {
        case "day":
            return {
                kind: "day",
                season: moment.value.season,
                number: moment.value.day,
                roundId: moment.value.roundId,
                endedAt: msOf(moment.value.endedAt),
            }
        case "battle":
            return battleEndedOf(moment.value)
        case "season":
            return {
                kind: "season",
                season: moment.value.season,
                bookId: moment.value.bookId,
                endedAt: msOf(moment.value.endedAt),
                finaleRoundId: moment.value.finaleRoundId,
                finaleBattle: moment.value.finaleBattle && battleEndedOf(moment.value.finaleBattle),
            }
        case undefined:
            return undefined
    }
}

function battleEndedOf(battle: BattleEndedMessage): BattleEnded {
    return {
        kind: "battle",
        season: battle.season,
        number: battle.number,
        entryId: battle.entryId,
        times: battle.times,
        endsSeason: battle.endsSeason,
        roundId: battle.roundId,
        endedAt: msOf(battle.endedAt),
    }
}

function msOf(at: Timestamp | undefined): number {
    return at?.toDate().getTime() ?? 0
}

function kindOf(kind: EntryKind): TurnKind {
    switch (kind) {
        case EntryKind.BATTLE:
            return "battle"
        case EntryKind.PLAIN:
            return "plain"
        case EntryKind.HALLOWEEN:
            return "halloween"
        case EntryKind.UNSPECIFIED:
            throw new Error("the turn has no kind")
        default:
            throw new Error(`the turn has a kind this build does not know: ${kind as number}`)
    }
}
