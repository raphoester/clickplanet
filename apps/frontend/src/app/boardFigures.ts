export type Figure = "tiles" | "share" | "points" | "today"

export const FIGURES: Record<Figure, string> = {
    tiles: "Tiles this country holds now",
    share: "Its share of the whole map now",
    points: "Season points. Each day, the 10 countries that held the most ground score. Most points wins the season",
    today: "What this country scores if the day ends now",
}
