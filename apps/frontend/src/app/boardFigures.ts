export const TAP_HINT_MS = 7000
export const GUIDE_MS = 10_000

export type Figure = "tiles" | "share" | "gained" | "gainedBattle" | "points" | "today" | "battle"

export const FIGURES: Record<Figure, string> = {
    tiles: "Tiles this country holds now, and its share of the map",
    share: "Its share of the whole map now",
    gained: "Tiles this country took since the day began, minus the tiles it lost. When the day ends, the 10 that gained the most score",
    gainedBattle: "Tiles this country took since the battle began, minus the tiles it lost. When it ends, the 10 that gained the most score",
    points: "Season points won on the days that ended. When a day ends, the 10 countries that gained the most ground that day score. Most points wins the season",
    today: "What this country scores if the day ends now. Only the ground gained since the day began counts",
    battle: "What this country scores if the battle ends now: a day's points, times the number shown. Only the ground gained since the battle began counts",
}
