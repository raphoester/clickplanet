import type {Fortification} from "../backends/backend.ts"

// Under this, another flag's fortify is a tiny island somewhere: it plays on the globe and says nothing.
export const NEWS_FROM_TILES = 50

export const FORTIFY_GUIDE_STORAGE_KEY = "clickplanet-fortify-guide"

export function isNews(fortification: Fortification, tiles: number, flag: string): boolean {
    return fortification.countryId === flag || tiles >= NEWS_FROM_TILES
}

export type CloseToFortify = {
    landmass: number
    name: string
    missing: number
}

export type Fortified = {
    fortification: Fortification
    name: string
    tiles: number
    news: boolean
}
