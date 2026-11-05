export const TAP_HINT_MS = 7000
export const GUIDE_MS = 10_000

export type Figure = "tiles" | "share" | "points" | "today" | "battle"

export const FIGURES: Record<Figure, string> = {
    tiles: "Tiles this country holds now",
    share: "Its share of the whole map now",
    points: "Season points won on the days that ended. When a day ends, the 10 countries that hold the most ground score. Most points wins the season",
    today: "What this country scores if the day ends now. Only the ground held when the day ends counts",
    battle: "What this country scores if the battle ends now: a day's points, times the number shown. Only the ground held when it ends counts",
}
