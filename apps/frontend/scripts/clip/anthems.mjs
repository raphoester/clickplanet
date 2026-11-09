import crypto from "node:crypto"
import fs from "node:fs"
import path from "node:path"
import {CACHE, encode} from "../anthemEncoding.mjs"

// Anthems the clips play and the game does not: the countries the US Navy Band never recorded, and Europe's, for a
// continent striking back. A recording that is not in the public domain carries the credit its licence asks for,
// which every caption it plays under repeats. Never the Royal Australian Navy Band's: its catalogue is for non-profit
// use only, whatever Commons tags them.
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
    ad: {
        file: "5/5b/El_Gran_Carlemany.ogg",
        title: "El Gran Carlemany",
    },
    bt: {
        file: "d/d4/National_Anthem_of_Bhutan_“Druk_tsendhen”.ogg",
        title: "Druk Tsendhen",
        credit: "Druk Tsendhen by Keith Terrett, CC BY 4.0, nationalanthems.info",
    },
    gd: {
        file: "7/76/HailGrenada.ogg",
        title: "Hail Grenada",
    },
    ne: {
        file: "1/10/THE_HONOR_OF_THE_FATHERLAND.ogg",
        title: "L'Honneur de la Patrie",
    },
    // CC BY-SA: a video with music synced to it is an adaptation, which the licence makes the video share alike.
    nr: {
        file: "a/a9/Nauru_Bwiema_instrumental.ogg",
        title: "Nauru Bwiema",
        credit: "Nauru Bwiema (instrumental) by Laurence Henry Hicks, nationalanthems.info. "
            + "This video is shared under CC BY-SA 3.0, as its music: creativecommons.org/licenses/by-sa/3.0",
    },
    sm: {
        file: "d/d9/Inno_Nazionale_della_Repubblica.ogg",
        title: "Inno Nazionale della Repubblica",
        credit: "Inno Nazionale della Repubblica (instrumental) by nationalanthems.info, CC BY 3.0",
    },
    to: {
        file: "d/d3/Fasifakafonua.ogg",
        title: "Ko e fasi ʻo e tuʻi ʻo e ʻOtu Tonga",
    },
    tw: {
        file: "b/be/National_anthem_of_the_Republic_of_China_(Taiwan)_中華民國國歌(演奏版).ogg",
        title: "Three Principles of the People",
        credit: "National Anthem of the Republic of China by its Ministry of Foreign Affairs, "
            + "Government Website Open Information Announcement",
    },
    ws: {
        file: "b/ba/SamoaAnthem.ogg",
        title: "The Banner of Freedom",
    },
    zm: {
        file: "f/f6/Zambian_national_anthem.oga",
        title: "Stand and Sing of Zambia, Proud and Free",
        credit: "Stand and Sing of Zambia (instrumental) by nationalanthems.info, CC BY 3.0",
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
