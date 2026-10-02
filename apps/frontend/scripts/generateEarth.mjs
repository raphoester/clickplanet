import fs from "node:fs"
import path from "node:path"
import {createHash} from "node:crypto"
import {fileURLToPath} from "node:url"

import sharp from "sharp"

import {readCoordinates} from "./map/blob.mjs"
import {coverage, spacingOf} from "./map/coverage.mjs"
import {DETAIL, lattice, neighbours} from "./map/lattice.mjs"
import {recolour} from "./map/recolour.mjs"

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..")
const earthDir = path.join(frontendRoot, "static", "earth")
const assetModule = path.join(frontendRoot, "src", "app", "viewer", "earthAsset.ts")

const QUALITY = 88
const NAME = /^earth-[0-9a-f]{8}\.jpg$/

const source = process.argv[2] ?? path.join(earthDir, "earth-source.jpg")
if (!fs.existsSync(source)) {
    throw new Error(`${source} is missing: it is the satellite mosaic the texture is cut from`)
}

const tiles = readCoordinates()
const {data: photo, info} = await sharp(source).raw().toBuffer({resolveWithObject: true})
const {width, height, channels} = info
console.log(`${path.basename(source)}: ${width}x${height}, ${tiles.name}: ${tiles.count} tiles`)

const {positions} = lattice(DETAIL)
const spacing = spacingOf(positions, neighbours(DETAIL))
console.log(`tile spacing ${(spacing * 6371).toFixed(1)} km, so a cell is about ${
    (spacing * 180 / Math.PI * height / 180 / Math.sqrt(3)).toFixed(2)} pixels across`)

const cover = coverage(tiles, spacing, width, height)
const land = cover.reduce((total, value) => total + (value > 0.5 ? 1 : 0), 0)
console.log(`the tile field covers ${(land / cover.length * 100).toFixed(2)}% of the image`)

const {pixels, moved} = recolour(photo, cover, {width, height, channels})
console.log(`${moved} of ${width * height} pixels moved (${(moved / (width * height) * 100).toFixed(2)}%)`)

const bytes = await sharp(pixels, {raw: {width, height, channels: 3}})
    .jpeg({quality: QUALITY, mozjpeg: true})
    .toBuffer()

const fileName = `earth-${createHash("sha256").update(bytes).digest("hex").slice(0, 8)}.jpg`
for (const entry of fs.readdirSync(earthDir)) {
    if (NAME.test(entry) && entry !== fileName) fs.unlinkSync(path.join(earthDir, entry))
}
fs.writeFileSync(path.join(earthDir, fileName), bytes)

fs.writeFileSync(assetModule, `export const EARTH_URL = "/static/earth/${fileName}"
`)

console.log(`wrote static/earth/${fileName}: ${(bytes.byteLength / 1024).toFixed(0)} KB`)
