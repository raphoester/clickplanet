import {existsSync, mkdtempSync, readdirSync, readFileSync, rmSync} from "node:fs"
import {tmpdir} from "node:os"
import {basename, join} from "node:path"
import sharp from "sharp"
import * as Smith from "spritesmith"
import {decodeBorders} from "../src/app/viewer/borderField.ts"
import type {Region} from "../src/app/viewer/atlas.ts"
import {writeAtlas} from "./writeAtlas.ts"

const FLAG_WIDTH = 200

const countriesDir = "static/countries"
const sources = ["svg", "custom"].map((dir) => join(countriesDir, dir))

const codes = [...new Set([...mapCodes(), ...countryCodes()])].sort()

const rendered = mkdtempSync(join(tmpdir(), "clickplanet-flags-"))
try {
    const files: string[] = []
    for (const code of codes) files.push(await rasterize(code, rendered))

    const packed = await pack(files)
    const coordinates: Record<string, Region> = Object.fromEntries(
        Object.entries(packed.coordinates).map(([file, region]) => [basename(file, ".png"), region]),
    )
    const image = await sharp(packed.image).webp({lossless: true, effort: 6}).toBuffer()

    const {fileName, width, height} = await writeAtlas(image, coordinates)
    console.log(`wrote ${countriesDir}/${fileName}: ${width}x${height}, ${codes.length} flags of ${FLAG_WIDTH}px, ${(image.length / 1024).toFixed(0)} kB`)
} finally {
    rmSync(rendered, {recursive: true, force: true})
}

function mapCodes(): string[] {
    const found = readdirSync("static").filter((entry) => /^borders-[0-9a-f]{8}\.bin$/.test(entry))
    if (found.length !== 1) throw new Error(`expected one static/borders-<hash>.bin, found ${found.length}`)
    const blob = readFileSync(join("static", found[0]))
    const {codes} = decodeBorders(blob.buffer.slice(blob.byteOffset, blob.byteOffset + blob.byteLength))
    return codes.filter((code) => code !== "")
}

function countryCodes(): string[] {
    return Object.keys(JSON.parse(readFileSync(join(countriesDir, "countries.json"), "utf8")))
}

function svgOf(code: string): string {
    for (const dir of sources) {
        const file = join(dir, `${code}.svg`)
        if (existsSync(file)) return file
    }
    throw new Error(`no flag for ${code}: put ${code}.svg in ${sources.join(" or ")}`)
}

async function rasterize(code: string, into: string): Promise<string> {
    const svg = svgOf(code)
    const {width} = await sharp(svg, {limitInputPixels: false}).metadata()
    if (!width) throw new Error(`${svg} has no width`)
    // Rendered a little wider than wanted and scaled down, never up.
    const density = Math.ceil(72 * FLAG_WIDTH / width) + 1
    const file = join(into, `${code}.png`)
    await sharp(svg, {density, limitInputPixels: false}).resize({width: FLAG_WIDTH}).png().toFile(file)
    return file
}

function pack(files: string[]): Promise<Smith.SpritesmithResult> {
    return new Promise((resolve, reject) => {
        Smith.default.run({src: files}, (err: Error | null, result: Smith.SpritesmithResult) => {
            if (err) reject(err)
            else resolve(result)
        })
    })
}
