import {LeaderboardEntry} from "./leaderboard.ts";

/**
 * How long a badge stays up after the last tile it counts. Long enough that a
 * run of clicks lands in the same badge and reads as one move, short enough
 * that a quiet leaderboard is quiet again.
 */
export const DELTA_HOLD_MS = 2_500

export type TileDelta = {
    /** Tiles won (or lost) while the badge has been up. Never zero. */
    net: number
    /** Bumped by every change the badge takes in, so the view can replay it. */
    beat: number
    /** When the last change landed. */
    at: number
}

export type TileDeltas = ReadonlyMap<string, TileDelta>

export const NO_TILE_DELTAS: TileDeltas = new Map()

export function tileCounts(entries: readonly LeaderboardEntry[]): ReadonlyMap<string, number> {
    return new Map(entries.map((entry) => [entry.country.code, entry.tiles]))
}

/**
 * Folds one leaderboard change into the badges already up. A country that keeps
 * winning tiles keeps the same badge, counting up, instead of flashing "+1"
 * three times over.
 */
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

        // A country that wins a tile back and loses it again has nothing left to
        // report, so its badge goes rather than reading "+0".
        if (net === 0) {
            next.delete(code)
            continue
        }

        next.set(code, {net, beat: (standing?.beat ?? 0) + 1, at: now})
    }

    return moved ? next : badges
}

/** Returns the badges it was given when none of them is due, so a quiet
 *  leaderboard costs no render. */
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
