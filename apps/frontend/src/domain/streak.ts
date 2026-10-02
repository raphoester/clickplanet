export const MIN_STREAK_SHOWN = 3

export function streakShown(days: number): boolean {
    return days >= MIN_STREAK_SHOWN
}
