import {Country} from "./countries.ts";
import {LeaderboardEntry} from "./leaderboard.ts";

/**
 * Everything about a shared image that is decided before a pixel is drawn: where
 * it points, what it says, and how big it comes out.
 */

export const SHARE_ORIGIN = "https://clickplanet.lol"

/** The query parameter the link preview is keyed on. Renaming it here renames it
 *  in the image, in the shared text, and in nothing the Worker reads — the two
 *  have to be changed together. */
export const SHARE_COUNTRY_PARAM = "c"

export type ShareStats = {
    country: Country
    /** `null` is a country holding no tile at all, which has no rank to show. */
    rank: number | null
    tiles: number
}

/* The copy around it is English, so the grouping is too — same reason as the
   leaderboard's own count. */
const grouped = new Intl.NumberFormat("en-US")

export function shareStats(leaderboard: readonly LeaderboardEntry[], country: Country): ShareStats {
    const index = leaderboard.findIndex((entry) => entry.country.code === country.code)
    if (index === -1) return {country, rank: null, tiles: 0}

    return {country, rank: index + 1, tiles: leaderboard[index].tiles}
}

export function shareUrl(code: string): string {
    return `${SHARE_ORIGIN}/?${SHARE_COUNTRY_PARAM}=${encodeURIComponent(code)}`
}

/** The same link with the scheme dropped, because it is *drawn into the image*:
 *  what travels is a picture, so the link has to be short enough to retype and
 *  is read by a person rather than followed by a browser. */
export function shareLabel(code: string): string {
    return shareUrl(code).replace(/^https?:\/\//, "")
}

export function shareFileName(code: string): string {
    return `clickplanet-${code}.png`
}

/** What the image says under the flag. Uppercase is the label style everywhere
 *  else on the card, so it is baked in rather than left to CSS that is not there. */
export function statsLine(stats: ShareStats): string {
    if (stats.rank === null) return "NO TILES YET"

    const tiles = stats.tiles === 1 ? "1 TILE" : `${grouped.format(stats.tiles)} TILES`
    return `RANK #${stats.rank} · ${tiles}`
}

/** What rides along with the image where the delivery takes text too. */
export function shareText(stats: ShareStats): string {
    return `${headline(stats)} ${shareUrl(stats.country.code)}`
}

function headline({country, rank, tiles}: ShareStats): string {
    if (rank === null) return `${country.name} holds nothing on ClickPlanet yet. Come and claim it.`

    return `${country.name} is #${rank} on ClickPlanet with ${grouped.format(tiles)} ` +
        `${tiles === 1 ? "tile" : "tiles"}. Come and take them.`
}

/** Under this, a shared image reads as a thumbnail wherever it is posted. */
const MIN_EDGE = 720

/** Over this, it is a few megabytes of PNG that a share sheet may refuse. */
const MAX_EDGE = 2400

/** The globe is captured at the size it is drawn, so anything past this is
 *  interpolation — it only sharpens the text and the flag laid over it. */
const MAX_SCALE = 2

/**
 * The size the card comes out at, from the size the globe was captured at.
 *
 * The canvas is sized in CSS pixels rather than device pixels, so a phone
 * captures around 390×844 — honest, and far too small to post. Scaling up
 * softens the globe a little and keeps the flag and the counts crisp, which is
 * the half of the image anyone reads.
 */
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

/** The biggest a sprite fits in a box without being stretched — the flags are
 *  every aspect ratio there is, and a squashed one is the wrong flag. */
export function fitInBox(
    source: {width: number, height: number},
    box: {width: number, height: number},
): {width: number, height: number} {
    if (source.width <= 0 || source.height <= 0) return {width: 0, height: 0}

    const scale = Math.min(box.width / source.width, box.height / source.height)
    return {width: source.width * scale, height: source.height * scale}
}
