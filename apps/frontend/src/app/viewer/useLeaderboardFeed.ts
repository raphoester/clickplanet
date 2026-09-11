import {useCallback, useEffect, useRef, useState} from "react";
import {LeaderboardEntry} from "../../domain/leaderboard.ts";
import {
    expireBadges,
    NO_TILE_DELTAS,
    takeInChanges,
    TileDeltas,
    tileCounts,
} from "../../domain/tileDeltas.ts";

/**
 * How often the board is published to React. The globe re-ranks on every batch
 * it takes in — ten times a second, over every country on the map — and
 * rendering each of those is what a busy planet cannot afford: a hundred rows
 * and a hundred badge animations restarted ten times a second is enough to make
 * the whole machine stutter.
 *
 * The accounting still follows every batch, because it is pure arithmetic over
 * a couple of hundred numbers; only the render is sampled. That costs at most
 * half a second of staleness on a count nobody reads that fast, and it is what
 * makes the badges legible: a country that won three tiles in half a second
 * says "+3" once, rather than "+1" three times too fast to read.
 */
export const SAMPLE_MS = 500

export type LeaderboardFeed = {
    leaderboard: LeaderboardEntry[]
    tileDeltas: TileDeltas
    recordLeaderboard: (entries: LeaderboardEntry[], live: boolean) => void
}

type Board = {
    entries: LeaderboardEntry[]
    deltas: TileDeltas
}

const EMPTY_BOARD: Board = {entries: [], deltas: NO_TILE_DELTAS}

export function useLeaderboardFeed(): LeaderboardFeed {
    const [board, setBoard] = useState<Board>(EMPTY_BOARD)

    // What the globe has said since the last tick. All refs: taking a board in
    // must not cost a render, which is the whole point of sampling.
    const latest = useRef<LeaderboardEntry[] | null>(null)
    const badges = useRef<TileDeltas>(NO_TILE_DELTAS)
    const counts = useRef<ReadonlyMap<string, number>>(new Map())

    const recordLeaderboard = useCallback((entries: LeaderboardEntry[], live: boolean) => {
        const before = counts.current
        const after = tileCounts(entries)
        counts.current = after

        // Every board is measured from the one before it, live or not — the map
        // as it loads is the ground the news lands on, and is not news itself.
        if (live) badges.current = takeInChanges(badges.current, before, after, Date.now())

        latest.current = entries
    }, [])

    useEffect(() => {
        const timer = setInterval(() => {
            const entries = latest.current
            latest.current = null

            const deltas = expireBadges(badges.current, Date.now())

            // A still board with nothing left to retire is left alone entirely:
            // a tick that hands React the state it already holds still costs a
            // render before it bails out.
            if (entries === null && deltas === badges.current) return
            badges.current = deltas

            setBoard((standing) =>
                ({entries: entries ?? standing.entries, deltas}))
        }, SAMPLE_MS)

        return () => clearInterval(timer)
    }, [])

    return {leaderboard: board.entries, tileDeltas: board.deltas, recordLeaderboard}
}
