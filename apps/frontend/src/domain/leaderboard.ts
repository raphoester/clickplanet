import {Country} from "./countries.ts";

export type LeaderboardEntry = {
    country: Country
    tiles: number
}

export function rankCountries(counts: ReadonlyMap<string, number>): LeaderboardEntry[] {
    const ranked: LeaderboardEntry[] = []

    // A flag this build does not know is ranked too: the server counted it, and the page names it by its code.
    counts.forEach((tiles, code) => {
        if (tiles > 0 && code !== "") ranked.push({country: {code}, tiles})
    })

    return ranked.sort((a, b) =>
        b.tiles - a.tiles || a.country.code.localeCompare(b.country.code))
}

export function rankOf(leaderboard: readonly LeaderboardEntry[], country: Country): number | null {
    const index = leaderboard.findIndex((entry) => entry.country.code === country.code)
    return index === -1 ? null : index + 1
}
