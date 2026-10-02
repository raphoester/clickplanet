import crypto from "node:crypto"
import fs from "node:fs"
import path from "node:path"

const COMMIT = "06121655d0e82f9cae6e7ba6feed4fa6fdbfc2a4"
const SOURCE = `https://raw.githubusercontent.com/googlefonts/noto-emoji/${COMMIT}/2D/svg/`
const CACHE = "node_modules/.cache/reactions"
const OUT = "static/reactions"
const ASSET = "src/app/chat/reactionsAsset.ts"

const REACTIONS = {
    LAUGH: ["1f602", "Laughing"],
    CLOWN: ["1f921", "Clown"],
    SKULL: ["1f480", "Skull"],
    FIRE: ["1f525", "Fire"],
    THUMBS_UP: ["1f44d", "Thumbs up"],
    THUMBS_DOWN: ["1f44e", "Thumbs down"],
    HEART: ["2764", "Heart"],
    CRY: ["1f62d", "Crying"],
    SCREAM: ["1f631", "Screaming"],
    COOL: ["1f60e", "Cool"],
    THINK: ["1f914", "Thinking"],
    POOP: ["1f4a9", "Poop"],
    PARTY: ["1f973", "Party"],
    MIND_BLOWN: ["1f92f", "Mind blown"],
    NERD: ["1f913", "Nerd"],
    EARTH: ["1f30d", "Earth"],
}

async function download(codePoint) {
    const cached = path.join(CACHE, COMMIT, `${codePoint}.svg`)
    if (fs.existsSync(cached)) return fs.readFileSync(cached)

    const response = await fetch(`${SOURCE}emoji_u${codePoint}.svg`)
    if (!response.ok) throw new Error(`${codePoint}: HTTP ${response.status}`)
    const bytes = Buffer.from(await response.arrayBuffer())
    fs.mkdirSync(path.dirname(cached), {recursive: true})
    fs.writeFileSync(cached, bytes)
    return bytes
}

const images = await Promise.all(Object.values(REACTIONS).map(([codePoint]) => download(codePoint)))

fs.rmSync(OUT, {recursive: true, force: true})
fs.mkdirSync(OUT, {recursive: true})

const lines = Object.entries(REACTIONS).map(([name, [, label]], index) => {
    const bytes = images[index]
    const hash = crypto.createHash("sha256").update(bytes).digest("hex").slice(0, 8)
    const file = `${name.toLowerCase().replaceAll("_", "-")}-${hash}.svg`
    fs.writeFileSync(path.join(OUT, file), bytes)
    console.log(`${name}  ${Math.round(bytes.length / 1024)}K`)
    return `    [Reaction.${name}, {url: ${JSON.stringify(`/static/reactions/${file}`)}, label: ${JSON.stringify(label)}}],`
})

fs.writeFileSync(ASSET, [
    "import {Reaction} from \"../../gen/grpc/chat/v1/chat_pb.ts\";",
    "",
    "export const REACTION_IMAGES: ReadonlyMap<Reaction, {url: string, label: string}> = new Map([",
    ...lines,
    "])",
    "",
].join("\n"))

const total = images.reduce((sum, bytes) => sum + bytes.length, 0)
console.log(`${lines.length} reactions, ${Math.round(total / 1024)} KB`)
