import {OwnershipsGetter} from "./backend.ts"
import {MapHistory} from "../domain/mapHistory.ts"
import {fakeHistoryOf} from "./fakeHistory.ts"
import {RecentHistorySource} from "./recentHistory.ts"

const STEP_MS = 30 * 60_000
const FRAMES = 49

// A made-up 24 hours ending on the fake map.
export class FakeRecentHistory implements RecentHistorySource {
    constructor(
        private readonly map: OwnershipsGetter,
        private readonly tiles: () => Promise<number>,
        private readonly now: () => number = Date.now,
    ) {
    }

    public async recentHistory(): Promise<MapHistory> {
        const final = new Map<number, string>()
        await this.map.getOwnerships([{start: 1, end: await this.tiles()}], ({bindings}) => {
            bindings.forEach((owner, tile) => final.set(tile, owner))
        })
        const until = Math.floor(this.now() / STEP_MS) * STEP_MS
        return fakeHistoryOf(final, until - 48 * STEP_MS, until, FRAMES)
    }
}
