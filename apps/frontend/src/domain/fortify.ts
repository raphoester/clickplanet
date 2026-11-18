import type {Fortification} from "../backends/backend.ts"
import {BoardScale, scaledCount} from "./boardScale.ts"

// Under this many tiles of earth-1, another flag's fortify is a tiny island somewhere: it plays on the globe and says
// nothing. As much ground on another board.
const NEWS_FROM_EARTH_1_TILES = 50

export function newsFromTiles(scale: BoardScale): number {
    return scaledCount(NEWS_FROM_EARTH_1_TILES, scale)
}

export const FORTIFY_GUIDE_STORAGE_KEY = "clickplanet-fortify-guide"

export function isNews(fortification: Fortification, tiles: number, flag: string, scale: BoardScale): boolean {
    return fortification.countryId === flag || tiles >= newsFromTiles(scale)
}

export type Fortified = {
    fortification: Fortification
    name: string
    tiles: number
    news: boolean
}
