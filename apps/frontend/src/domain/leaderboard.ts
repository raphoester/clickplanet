import {Countries, Country} from "./countries.ts";
import {warnOnce} from "./warnOnce.ts";

export type LeaderboardEntry = {
    country: Country
    tiles: number
}

/**
 * Ranks the per-country tile counts for display.
 *
 * This used to be a stateful class that tracked its own totals with a linear
 * scan per entry, incremented and decremented them from update events, and
 * could be driven below zero when an event disagreed with what it had seen.
 * TileOwnership owns the counts now, so ranking is a pure function of them.
 */
export function rankCountries(counts: ReadonlyMap<string, number>): LeaderboardEntry[] {
    const ranked: LeaderboardEntry[] = []

    counts.forEach((tiles, code) => {
        if (tiles <= 0) return

        const country = Countries.get(code)
        if (!country) {
            warnOnce(`Leaving country "${code}" out of the leaderboard: no such country`)
            return
        }

        ranked.push({country, tiles})
    })

    /** Ties break on code so that equally-placed countries stop swapping rows. */
    return ranked.sort((a, b) =>
        b.tiles - a.tiles || a.country.code.localeCompare(b.country.code))
}

/**
 * Where a country sits in the ranking, 1-based, or null when it holds no tile.
 *
 * The menu keeps this on screen even when the card is folded away, so it reads
 * the rank off the same ordered list the table renders rather than deriving its
 * own — two answers that disagreed would be worse than none.
 */
export function rankOf(leaderboard: readonly LeaderboardEntry[], country: Country): number | null {
    const index = leaderboard.findIndex((entry) => entry.country.code === country.code)
    return index === -1 ? null : index + 1
}
