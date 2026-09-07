/**
 * Builds the social preview image from the raw screenshot.
 *
 *   npm run og-image
 *
 * Scrapers want 1.91:1; the screenshot is 2.2:1 with the leaderboard against
 * the left edge and the buttons against the right, so cropping to fit would eat
 * both. It is letterboxed onto black instead, which is invisible against the
 * game's own background.
 *
 * JPEG rather than PNG: the alpha channel is unused, LinkedIn and Slack handle
 * baseline JPEG most reliably, and it takes the file from ~940 kB to ~100 kB.
 */
import {readFileSync, writeFileSync} from "node:fs"
import {dirname, join, resolve} from "node:path"
import {fileURLToPath} from "node:url"
import sharp from "sharp"

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const staticDir = join(frontendRoot, "static")

/** What every scraper documents as the preferred size. */
export const OG_IMAGE_WIDTH = 1200
export const OG_IMAGE_HEIGHT = 627

const source = readFileSync(join(staticDir, "og-source.png"))

const image = await sharp(source)
    .resize(OG_IMAGE_WIDTH, OG_IMAGE_HEIGHT, {
        fit: "contain",
        background: {r: 0, g: 0, b: 0, alpha: 1},
    })
    .flatten({background: {r: 0, g: 0, b: 0}})
    .jpeg({quality: 86, mozjpeg: true, progressive: false})
    .toBuffer()

writeFileSync(join(staticDir, "og-image.jpg"), image)

console.log(
    `wrote static/og-image.jpg: ${OG_IMAGE_WIDTH}x${OG_IMAGE_HEIGHT}, ` +
    `${(image.length / 1024).toFixed(0)} kB (from ${(source.length / 1024).toFixed(0)} kB)`,
)
