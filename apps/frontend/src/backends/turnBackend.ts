import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {GamemasterService} from "../gen/grpc/gamemaster/v1/gamemaster_connect.ts"
import {
    EntryKind,
    GamemasterEvent,
    Season as SeasonMessage,
    Timeline,
    Turn as TurnMessage,
} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {Season, Seasons} from "./season.ts"
import {Config, NO_TIMEOUT, openStream, pageWakeups, Wakeups} from "./transport.ts"
import {BattleToCome, PLAIN_TURN, Turn, TurnBackend, TurnKind, TurnListener} from "./turn.ts"

export function newGamemasterServiceClient(config: Config): PromiseClient<typeof GamemasterService> {
    return createPromiseClient(GamemasterService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class ConnectTurnBackend implements TurnBackend {
    constructor(
        private readonly client: PromiseClient<typeof GamemasterService>,
        private readonly wakeups: Wakeups = pageWakeups,
    ) {
    }

    public follow({onTurn, onBattles, onSeasons}: TurnListener): () => void {
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
                        onSeasons({})
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
}

export function turnOf(turn: TurnMessage): Turn {
    return {
        kind: kindOf(turn.kind),
        since: turn.since?.toDate().getTime(),
        endsAt: turn.endsAt?.toDate().getTime(),
        description: turn.description,
        entryId: turn.entryId,
    }
}

export function battlesOf(timeline: Timeline): BattleToCome[] {
    return timeline.battles.flatMap((battle) => battle.startsAt && battle.endsAt
        ? [{
            entryId: battle.entryId,
            startsAt: battle.startsAt.toDate().getTime(),
            endsAt: battle.endsAt.toDate().getTime(),
            description: battle.description,
        }]
        : [])
}

export function seasonsOf(timeline: Timeline): Seasons {
    return {current: seasonOf(timeline.current), next: seasonOf(timeline.next)}
}

function seasonOf(season: SeasonMessage | undefined): Season | undefined {
    if (!season?.startsAt || !season.endsAt) return undefined
    return {number: season.number, startsAt: season.startsAt.toDate().getTime(), endsAt: season.endsAt.toDate().getTime()}
}

function kindOf(kind: EntryKind): TurnKind {
    switch (kind) {
        case EntryKind.BATTLE:
            return "battle"
        case EntryKind.PLAIN:
            return "plain"
        case EntryKind.UNSPECIFIED:
            throw new Error("the turn has no kind")
        default:
            throw new Error(`the turn has a kind this build does not know: ${kind as number}`)
    }
}
