import {useCallback, useEffect, useRef, useState} from "react";
import {LeaderboardEntry} from "../../domain/leaderboard.ts";
import {
    expireBadges,
    NO_TILE_DELTAS,
    takeInChanges,
    TileDeltas,
    tileCounts,
} from "../../domain/tileDeltas.ts";

export const SAMPLE_MS = 500

export type LeaderboardFeed = {
    leaderboard: LeaderboardEntry[]
    tileDeltas: TileDeltas
    recordLeaderboard: (entries: LeaderboardEntry[], live: boolean) => void
    publishLeaderboard: () => void
}

type Board = {
    entries: LeaderboardEntry[]
    deltas: TileDeltas
}

const EMPTY_BOARD: Board = {entries: [], deltas: NO_TILE_DELTAS}

export function useLeaderboardFeed(): LeaderboardFeed {
    const [board, setBoard] = useState<Board>(EMPTY_BOARD)

    const latest = useRef<LeaderboardEntry[] | null>(null)
    const badges = useRef<TileDeltas>(NO_TILE_DELTAS)
    const counts = useRef<ReadonlyMap<string, number>>(new Map())

    const recordLeaderboard = useCallback((entries: LeaderboardEntry[], live: boolean) => {
        const before = counts.current
        const after = tileCounts(entries)
        counts.current = after

        if (live) badges.current = takeInChanges(badges.current, before, after, Date.now())

        latest.current = entries
    }, [])

    const publish = useCallback(() => {
        const entries = latest.current
        latest.current = null

        const deltas = expireBadges(badges.current, Date.now())

        if (entries === null && deltas === badges.current) return
        badges.current = deltas

        setBoard((standing) =>
            ({entries: entries ?? standing.entries, deltas}))
    }, [])

    useEffect(() => {
        const timer = setInterval(publish, SAMPLE_MS)

        return () => clearInterval(timer)
    }, [publish])

    return {leaderboard: board.entries, tileDeltas: board.deltas, recordLeaderboard, publishLeaderboard: publish}
}
