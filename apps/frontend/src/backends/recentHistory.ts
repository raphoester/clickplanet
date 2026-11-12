import {PromiseClient} from "@connectrpc/connect"
import {ClickService} from "../gen/grpc/planet/v1/planet_connect.ts"
import {MapHistory as MapHistoryMessage} from "../gen/grpc/planet/v1/history_pb.ts"
import {RecentHistory} from "../domain/video/story.ts"
import {historyOf} from "./turnBackend.ts"

// About 830 KB on a phone's network.
const RECENT_HISTORY_TIMEOUT_MS = 60_000

export interface RecentHistorySource {
    recentHistory(): Promise<RecentHistory>
}

// A GET the browser's cache serves again until the next step.
export class ConnectRecentHistory implements RecentHistorySource {
    constructor(private readonly client: PromiseClient<typeof ClickService>) {
    }

    public async recentHistory(): Promise<RecentHistory> {
        const res = await this.client.getRecentMapHistory({}, {timeoutMs: RECENT_HISTORY_TIMEOUT_MS})
        if (!res.history) throw new Error("planet answered no history")
        return recentHistoryOf(res.history)
    }
}

export function recentHistoryOf(history: MapHistoryMessage): RecentHistory {
    const {frames, hits} = historyOf(history)
    return {
        frames: frames.map((frame, i) => ({...frame, accounts: history.frames[i].owners.map((owner) => owner.accountId)})),
        hits,
    }
}
