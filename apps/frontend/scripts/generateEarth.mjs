import fs from "node:fs"
import path from "node:path"
import {createHash} from "node:crypto"
import {fileURLToPath} from "node:url"

import sharp from "sharp"
import {encodeToKTX2} from "ktx2-encoder"

import {readCoordinates} from "./map/blob.mjs"
import {coverage, spacingOf, tilesOn} from "./map/coverage.mjs"
import {DETAIL, lattice, neighbours} from "./map/lattice.mjs"
import {recolour} from "./map/recolour.mjs"

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..")
const earthDir = path.join(frontendRoot, "static", "earth")
const assetModule = path.join(frontendRoot, "src", "app", "viewer", "earthAsset.ts")

const QUALITY = 80

const source = process.argv[2] ?? path.join(earthDir, "earth-source.jpg")
if (!fs.existsSync(source)) {
    throw new Error(`${source} is missing: it is the satellite mosaic the texture is cut from`)
}

const tiles = readCoordinates()
const {data: photo, info} = await sharp(source).raw().toBuffer({resolveWithObject: true})
const {width, height, channels} = info
console.log(`${path.basename(source)}: ${width}x${height}, ${tiles.name}: ${tiles.count} tiles`)

const vertices = lattice(DETAIL)
const spacing = spacingOf(vertices.positions, neighbours(DETAIL))
console.log(`tile spacing ${(spacing * 6371).toFixed(1)} km, so a cell is about ${
    (spacing * 180 / Math.PI * height / 180 / Math.sqrt(3)).toFixed(2)} pixels across`)

const cover = coverage(vertices, tilesOn(vertices, tiles), spacing, width, height)
const land = cover.reduce((total, value) => total + (value > 0.5 ? 1 : 0), 0)
console.log(`the tile field covers ${(land / cover.length * 100).toFixed(2)}% of the image`)

const {pixels, moved} = recolour(photo, cover, {width, height, channels})
console.log(`${moved} of ${width * height} pixels moved (${(moved / (width * height) * 100).toFixed(2)}%)`)

const jpeg = await sharp(pixels, {raw: {width, height, channels: 3}})
    .jpeg({quality: QUALITY, mozjpeg: true, chromaSubsampling: "4:4:4"})
    .toBuffer()

const rgba = await sharp(pixels, {raw: {width, height, channels: 3}}).ensureAlpha().raw().toBuffer()
const compressed = (options) => encodeToKTX2(rgba, {
    generateMipmap: true,
    isYFlip: true,
    isPerceptual: true,
    // The JPEG's bytes are sampled as they are, and both earth shaders are tuned to that.
    isSetKTX2SRGBTransferFunc: false,
    imageDecoder: async (data) => ({width, height, data}),
    ...options,
})
const etc1s = await compressed({isUASTC: false, qualityLevel: 255})
const uastc = await compressed({isUASTC: true, uastcLDRQualityLevel: 2, needSupercompression: true})

const jpegName = write("earth", "jpg", jpeg)
const etc1sName = write("earth-etc1s", "ktx2", etc1s)
const uastcName = write("earth-uastc", "ktx2", uastc)

fs.writeFileSync(assetModule, `export const EARTH_URL = "/static/earth/${jpegName}"
export const EARTH_ETC1S_URL = "/static/earth/${etc1sName}"
export const EARTH_UASTC_URL = "/static/earth/${uastcName}"
`)

function write(prefix, extension, bytes) {
    const name = new RegExp(`^${prefix}-[0-9a-f]{8}\\.${extension}$`)
    const fileName = `${prefix}-${createHash("sha256").update(bytes).digest("hex").slice(0, 8)}.${extension}`
    for (const entry of fs.readdirSync(earthDir)) {
        if (name.test(entry) && entry !== fileName) fs.unlinkSync(path.join(earthDir, entry))
    }
    fs.writeFileSync(path.join(earthDir, fileName), bytes)
    console.log(`wrote static/earth/${fileName}: ${(bytes.byteLength / 1024).toFixed(0)} KB`)
    return fileName
}
