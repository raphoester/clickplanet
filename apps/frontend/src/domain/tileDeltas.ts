import {LeaderboardEntry} from "./leaderboard.ts";

export const DELTA_HOLD_MS = 2_500

export type TileDelta = {
    net: number
    beat: number
    at: number
}

export type TileDeltas = ReadonlyMap<string, TileDelta>

export const NO_TILE_DELTAS: TileDeltas = new Map()

export function tileCounts(entries: readonly LeaderboardEntry[]): ReadonlyMap<string, number> {
    return new Map(entries.map((entry) => [entry.country.code, entry.tiles]))
}

export function takeInChanges(
    badges: TileDeltas,
    before: ReadonlyMap<string, number>,
    after: ReadonlyMap<string, number>,
    now: number,
): TileDeltas {
    const next = new Map(badges)
    let moved = false

    for (const code of new Set([...before.keys(), ...after.keys()])) {
        const change = (after.get(code) ?? 0) - (before.get(code) ?? 0)
        if (change === 0) continue

        moved = true

        const standing = next.get(code)
        const net = (standing?.net ?? 0) + change

        if (net === 0) {
            next.delete(code)
            continue
        }

        next.set(code, {net, beat: (standing?.beat ?? 0) + 1, at: now})
    }

    return moved ? next : badges
}

export function expireBadges(badges: TileDeltas, now: number): TileDeltas {
    const kept = new Map<string, TileDelta>()
    badges.forEach((badge, code) => {
        if (now - badge.at < DELTA_HOLD_MS) kept.set(code, badge)
    })
    return kept.size === badges.size ? badges : kept
}

export function signed(net: number): string {
    return net > 0 ? `+${net}` : `${net}`
}
