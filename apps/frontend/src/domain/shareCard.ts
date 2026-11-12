import {Countries, Country} from "./countries.ts";
import {LeaderboardEntry} from "./leaderboard.ts";

const SHARE_ORIGIN = "https://clickplanet.lol"

export const SHARE_FLAG_PARAM = "f"

export type ShareAddress = {
    code: string
    name?: string
}

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

// Every share's address is made here: the name it carries is the player's username, and only once the clips let a
// video show that very name (shown, as GetShownNames answers it); none for a guest.
export function shareAddress(code: string, username: string | undefined, shown: string | undefined): ShareAddress {
    return username && shown === username ? {code, name: username} : {code}
}

export function shareUrl(address: ShareAddress): string {
    return `${SHARE_ORIGIN}/${pathOf(address).map(encodeURIComponent).join("/")}`
}

function pathOf({code, name}: ShareAddress): string[] {
    // A username holds no "-": each of its spaces is one, since a space would end the link in a message.
    return name ? [code, name.replaceAll(" ", "-")] : [code]
}

export function sharedCountry(url: URL): Country | undefined {
    const code = url.searchParams.get(SHARE_FLAG_PARAM)
    return code ? Countries.get(code.trim().toLowerCase()) : undefined
}

// As a person reads and retypes it: no scheme, and the name in its own letters.
export function shareLabel(address: ShareAddress): string {
    return [SHARE_ORIGIN.replace(/^https?:\/\//, ""), ...pathOf(address)].join("/")
}

export type LinkPlacement = {beside: boolean, scale: number}

export function linkPlacement(link: number, beside: number, line: number): LinkPlacement {
    if (link <= beside) return {beside: true, scale: 1}
    return {beside: false, scale: Math.min(1, line / link)}
}

export function withoutSharedFlag(url: URL): URL {
    const stripped = new URL(url)
    stripped.searchParams.delete(SHARE_FLAG_PARAM)
    return stripped
}

export function shareFileName(code: string): string {
    return `clickplanet-${code}.png`
}

export function statsLine(stats: ShareStats): string {
    if (stats.rank === null) return "NO TILES YET"

    return `RANK #${stats.rank} · ${tilesHeld(stats.tiles).toUpperCase()}`
}

export function shareText(stats: ShareStats, address: ShareAddress): string {
    return `${shareHeadline(stats)} ${shareUrl(address)}`
}

export function breakHeadline(country: Country, nextSeason?: number): string {
    const join = `Play for ${country.name} on ClickPlanet!`
    return nextSeason === undefined ? join : `Season ${nextSeason} starts on a blank map. ${join}`
}

export function shareHeadline({country, rank, tiles}: ShareStats): string {
    if (rank === null) return `${country.name} holds nothing on ClickPlanet yet. Come and claim it.`

    return `${country.name} is #${rank} on ClickPlanet with ${tilesHeld(tiles)}. ` +
        (tiles === 0 ? "Come and claim it." : "Come and take them.")
}

function tilesHeld(tiles: number): string {
    if (tiles === 0) return "no tiles"
    return `${grouped.format(tiles)} ${tiles === 1 ? "tile" : "tiles"}`
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
