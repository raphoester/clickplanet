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
