import {PromiseClient} from "@connectrpc/connect"
import {ClickService} from "../gen/grpc/planet/v1/planet_connect.ts"
import {MapHistory} from "../domain/mapHistory.ts"
import {historyOf} from "./turnBackend.ts"

// About 800 KB on a phone's network.
const RECENT_HISTORY_TIMEOUT_MS = 60_000

export interface RecentHistorySource {
    recentHistory(): Promise<MapHistory>
}

// A GET the browser's cache serves again until the next step.
export class ConnectRecentHistory implements RecentHistorySource {
    constructor(private readonly client: PromiseClient<typeof ClickService>) {
    }

    public async recentHistory(): Promise<MapHistory> {
        const res = await this.client.getRecentMapHistory({}, {timeoutMs: RECENT_HISTORY_TIMEOUT_MS})
        if (!res.history) throw new Error("planet answered no history")
        return historyOf(res.history)
    }
}
