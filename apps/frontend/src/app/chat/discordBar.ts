export const DISCORD_BAR_STORAGE_KEY = "clickplanet-discord-bar"

// Under a day: a player who comes back at the same hour finds the bar again.
export const DISCORD_BAR_RETURNS_AFTER_MS = 20 * 3600_000

export function isStillHidden(hiddenAt: string | null, now: number): boolean {
    if (!hiddenAt) return false

    const since = now - Number(hiddenAt)
    return since >= 0 && since < DISCORD_BAR_RETURNS_AFTER_MS
}
