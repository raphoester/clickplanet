import {OwnershipsGetter} from "./backend.ts"
import {RecentHistory} from "../domain/video/story.ts"
import {fakeHistoryOf} from "./fakeHistory.ts"
import {FAKE_TAKERS} from "./fakeAccounts.ts"
import {RecentHistorySource} from "./recentHistory.ts"

const STEP_MS = 30 * 60_000
const FRAMES = 49

// A made-up 24 hours ending on the fake map, each change made by one of the fake takers.
export class FakeRecentHistory implements RecentHistorySource {
    constructor(
        private readonly map: OwnershipsGetter,
        private readonly tiles: () => Promise<number>,
        private readonly now: () => number = Date.now,
    ) {
    }

    public async recentHistory(): Promise<RecentHistory> {
        const final = new Map<number, string>()
        await this.map.getOwnerships([{start: 1, end: await this.tiles()}], ({bindings}) => {
            bindings.forEach((owner, tile) => final.set(tile, owner))
        })
        const until = Math.floor(this.now() / STEP_MS) * STEP_MS
        const {frames, hits} = fakeHistoryOf(final, until - 48 * STEP_MS, until, FRAMES)
        return {
            frames: frames.map((frame, k) => ({
                ...frame,
                accounts: Array.from(frame.tiles, (tile) => k === 0 ? "" : FAKE_TAKERS[tile % FAKE_TAKERS.length]),
            })),
            hits,
        }
    }
}
