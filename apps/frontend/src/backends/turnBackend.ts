import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {GamemasterService} from "../gen/grpc/gamemaster/v1/gamemaster_connect.ts"
import {EntryKind, GetTurnResponse} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {Config, pageWakeups, retrying, Wakeups} from "./transport.ts"
import {PLAIN_TURN, Turn, TurnBackend, TurnKind} from "./turn.ts"

export function newGamemasterServiceClient(config: Config): PromiseClient<typeof GamemasterService> {
    return createPromiseClient(GamemasterService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class ConnectTurnBackend implements TurnBackend {
    constructor(
        private readonly client: PromiseClient<typeof GamemasterService>,
        private readonly wakeups: Wakeups = pageWakeups,
    ) {
    }

    public async turn(): Promise<Turn> {
        try {
            return turnOf(await retrying(() => this.client.getTurn({}), "GetTurn"))
        } catch (e) {
            if (e instanceof ConnectError && (e.code === Code.Unimplemented || e.code === Code.NotFound)) return PLAIN_TURN
            throw e
        }
    }

    public changes(onChange: () => void): () => void {
        return this.wakeups(onChange)
    }
}

export function turnOf(res: GetTurnResponse): Turn {
    const turn = res.turn
    return {
        kind: kindOf(turn?.kind ?? EntryKind.UNSPECIFIED),
        since: turn?.since?.toDate().getTime(),
        endsAt: turn?.endsAt?.toDate().getTime(),
        description: turn?.description ?? "",
        entryId: turn?.entryId ?? "",
    }
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
