import {Countries, Country} from "./countries.ts";
import {warnOnce} from "./warnOnce.ts";

export type LeaderboardEntry = {
    country: Country
    tiles: number
}

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

    return ranked.sort((a, b) =>
        b.tiles - a.tiles || a.country.code.localeCompare(b.country.code))
}

export function rankOf(leaderboard: readonly LeaderboardEntry[], country: Country): number | null {
    const index = leaderboard.findIndex((entry) => entry.country.code === country.code)
    return index === -1 ? null : index + 1
}
