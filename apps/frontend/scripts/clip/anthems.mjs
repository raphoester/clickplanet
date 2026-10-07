import crypto from "node:crypto"
import fs from "node:fs"
import path from "node:path"
import {CACHE, encode} from "../anthemEncoding.mjs"

// Anthems the clips play and the game does not: Palestine's, which the US Navy Band never recorded, and Europe's,
// for a continent striking back. A recording that is not in the public domain carries the credit its licence asks
// for, which every caption it plays under repeats.
const OUT = "scripts/clip/anthems"
const ASSET = "src/clip/clipAnthemsAsset.ts"
const COMMONS = "https://upload.wikimedia.org/wikipedia/commons/"

const RECORDINGS = {
    eu: {
        file: "f/f3/Anthem_of_Europe_(US_Navy_instrumental_short_version).ogg",
        title: "Anthem of Europe",
    },
    ps: {
        file: "e/eb/National_Anthem_of_Palestine_-_Fida'i_-_Instrumental_فِدَائِي_فِدَائِي_فِدَائِي.ogg",
        title: "Fida'i",
        credit: "Fida'i (instrumental) by Sounds Monarch, CC BY 3.0, youtube.com/watch?v=2TszL-Fh6D8",
    },
}

async function download(code, file) {
    const cached = path.join(CACHE, `clip-${code}.ogg`)
    if (fs.existsSync(cached)) return cached

    const response = await fetch(COMMONS + encodeURI(file), {headers: {"User-Agent": "clickplanet-clips/1.0 (https://clickplanet.lol)"}})
    if (!response.ok) throw new Error(`${code}: HTTP ${response.status}`)
    fs.mkdirSync(path.dirname(cached), {recursive: true})
    fs.writeFileSync(cached, Buffer.from(await response.arrayBuffer()))
    return cached
}

fs.rmSync(OUT, {recursive: true, force: true})
fs.mkdirSync(OUT, {recursive: true})

const lines = []
for (const [code, {file, title, credit}] of Object.entries(RECORDINGS)) {
    const bytes = encode(await download(code, file))
    const hash = crypto.createHash("sha256").update(bytes).digest("hex").slice(0, 8)
    const name = `${code}-${hash}.m4a`
    fs.writeFileSync(path.join(OUT, name), bytes)
    const fields = [`url: ${JSON.stringify(`/${OUT}/${name}`)}`, `title: ${JSON.stringify(title)}`]
    if (credit) fields.push(`credit: ${JSON.stringify(credit)}`)
    lines.push(`    ${JSON.stringify(code)}: {${fields.join(", ")}},`)
    console.log(`${code}  ${title}  ${Math.round(bytes.length / 1024)}K`)
}

fs.writeFileSync(ASSET, [
    "export const CLIP_ANTHEMS: Readonly<Record<string, {url: string, title: string, credit?: string}>> = {",
    ...lines,
    "}",
    "",
].join("\n"))
