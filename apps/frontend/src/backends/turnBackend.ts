import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {MetagameService} from "../gen/grpc/metagame/v1/metagame_connect.ts"
import {EntryKind, GetTurnResponse} from "../gen/grpc/metagame/v1/metagame_pb.ts"
import {Config, pageWakeups, retrying, Wakeups} from "./transport.ts"
import {PLAIN_TURN, Turn, TurnBackend, TurnKind} from "./turn.ts"

export function newMetagameServiceClient(config: Config): PromiseClient<typeof MetagameService> {
    return createPromiseClient(MetagameService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class ConnectTurnBackend implements TurnBackend {
    constructor(
        private readonly client: PromiseClient<typeof MetagameService>,
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
    return {
        kind: kindOf(res.kind),
        since: res.since?.toDate().getTime(),
        endsAt: res.endsAt?.toDate().getTime(),
        description: "",
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
