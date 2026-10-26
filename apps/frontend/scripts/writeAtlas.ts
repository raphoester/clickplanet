import {createHash} from "node:crypto"
import {mkdirSync, readdirSync, unlinkSync, writeFileSync} from "node:fs"
import {dirname, join, resolve} from "node:path"
import {fileURLToPath} from "node:url"
import sharp from "sharp"
import type {Region} from "../src/app/viewer/atlas.ts"

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..")

const atlasDir = join(frontendRoot, "static", "countries")
const assetModulePath = join(frontendRoot, "src", "app", "viewer", "atlasAsset.ts")
const atlasNamePattern = /^atlas-[0-9a-f]{8}\.(png|webp)$/

// The smallest MAX_TEXTURE_SIZE among the phones still playing; three.js shrinks a bigger image in silence.
export const MAX_ATLAS_SIDE = 4096

export async function writeAtlas(image: Buffer, coordinates: Record<string, Region>): Promise<{
    fileName: string
    width: number
    height: number
}> {
    const {width, height} = await sharp(image).metadata()
    if (width === undefined || height === undefined) throw new Error("the atlas image has no size")
    if (width > MAX_ATLAS_SIDE || height > MAX_ATLAS_SIDE) {
        throw new Error(`the atlas is ${width}x${height}, over ${MAX_ATLAS_SIDE} a side: smaller flags, or fewer`)
    }
    for (const [code, region] of Object.entries(coordinates)) {
        if (region.x + region.width > width || region.y + region.height > height) {
            throw new Error(`${code}'s region falls outside the ${width}x${height} atlas`)
        }
    }

    const hash = createHash("sha256").update(image).digest("hex").slice(0, 8)
    const fileName = `atlas-${hash}.webp`

    mkdirSync(atlasDir, {recursive: true})
    for (const entry of readdirSync(atlasDir)) {
        if (atlasNamePattern.test(entry) && entry !== fileName) {
            unlinkSync(join(atlasDir, entry))
        }
    }

    writeFileSync(join(atlasDir, fileName), image)
    writeFileSync(join(atlasDir, "atlas.json"), JSON.stringify(coordinates, null, 2))
    writeFileSync(assetModulePath, assetModule(fileName, width, height))

    return {fileName, width, height}
}

function assetModule(fileName: string, width: number, height: number): string {
    return `export const ATLAS_URL = "/static/countries/${fileName}"

export const ATLAS_SIZE = {width: ${width}, height: ${height}} as const
`
}
