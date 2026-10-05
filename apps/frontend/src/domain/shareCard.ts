import {Countries, Country} from "./countries.ts";
import {LeaderboardEntry} from "./leaderboard.ts";

export const SHARE_ORIGIN = "https://clickplanet.lol"

export const SHARE_FLAG_PARAM = "f"

export type ShareStats = {
    country: Country
    rank: number | null
    tiles: number
}

const grouped = new Intl.NumberFormat("en-US")

export function shareStats(leaderboard: readonly LeaderboardEntry[], country: Country): ShareStats {
    const index = leaderboard.findIndex((entry) => entry.country.code === country.code)
    if (index === -1) return {country, rank: null, tiles: 0}

    return {country, rank: index + 1, tiles: leaderboard[index].tiles}
}

export function shareUrl(code: string): string {
    return `${SHARE_ORIGIN}/?${SHARE_FLAG_PARAM}=${encodeURIComponent(code)}`
}

export function sharedCountry(url: URL): Country | undefined {
    const code = url.searchParams.get(SHARE_FLAG_PARAM)
    return code ? Countries.get(code.trim().toLowerCase()) : undefined
}

export function withoutSharedFlag(url: URL): URL {
    const stripped = new URL(url)
    stripped.searchParams.delete(SHARE_FLAG_PARAM)
    return stripped
}

export function shareLabel(code: string): string {
    return shareUrl(code).replace(/^https?:\/\//, "")
}

export function shareFileName(code: string): string {
    return `clickplanet-${code}.png`
}

export function statsLine(stats: ShareStats): string {
    if (stats.rank === null) return "NO TILES YET"

    const tiles = stats.tiles === 1 ? "1 TILE" : `${grouped.format(stats.tiles)} TILES`
    return `RANK #${stats.rank} · ${tiles}`
}

export function shareText(stats: ShareStats): string {
    return `${headline(stats)} ${shareUrl(stats.country.code)}`
}

function headline({country, rank, tiles}: ShareStats): string {
    if (rank === null) return `${country.name} holds nothing on ClickPlanet yet. Come and claim it.`

    return `${country.name} is #${rank} on ClickPlanet with ${grouped.format(tiles)} ` +
        `${tiles === 1 ? "tile" : "tiles"}. Come and take them.`
}

const MIN_ASPECT = 9 / 16
const MAX_ASPECT = 16 / 9

const MIN_EDGE = 720

const MAX_EDGE = 2400

const MAX_SCALE = 2

export type Crop = {x: number, y: number, width: number, height: number}

export type CardLayout = {
    crop: Crop
    width: number
    height: number
}

export function cardLayout(width: number, height: number): CardLayout {
    const crop = cropToAspect(width, height)
    return {crop, ...cardSize(crop.width, crop.height)}
}

export function cropToAspect(width: number, height: number): Crop {
    if (width <= 0 || height <= 0) return {x: 0, y: 0, width, height}

    const aspect = width / height

    if (aspect < MIN_ASPECT) {
        const kept = Math.round(width / MIN_ASPECT)
        return {x: 0, y: Math.round((height - kept) / 2), width, height: kept}
    }

    if (aspect > MAX_ASPECT) {
        const kept = Math.round(height * MAX_ASPECT)
        return {x: Math.round((width - kept) / 2), y: 0, width: kept, height}
    }

    return {x: 0, y: 0, width, height}
}

export function cardSize(width: number, height: number): {width: number, height: number} {
    const scale = cardScale(width, height)
    return {width: Math.round(width * scale), height: Math.round(height * scale)}
}

export function cardScale(width: number, height: number): number {
    if (width <= 0 || height <= 0) return 1

    return Math.min(
        Math.max(1, MIN_EDGE / Math.min(width, height)),
        MAX_SCALE,
        MAX_EDGE / Math.max(width, height),
    )
}

export function fitInBox(
    source: {width: number, height: number},
    box: {width: number, height: number},
): {width: number, height: number} {
    if (source.width <= 0 || source.height <= 0) return {width: 0, height: 0}

    const scale = Math.min(box.width / source.width, box.height / source.height)
    return {width: source.width * scale, height: source.height * scale}
}
