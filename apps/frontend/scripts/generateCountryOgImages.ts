import {mkdirSync, readFileSync, rmSync, writeFileSync} from "node:fs"
import {dirname, join, resolve} from "node:path"
import {fileURLToPath} from "node:url"
import sharp from "sharp"
import {CARD_DIRECTORY, CARD_HEIGHT, CARD_WIDTH, ShareableCountries} from "../src/domain/shareCard.ts"

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const staticDir = join(frontendRoot, "static")
const cardDir = join(frontendRoot, CARD_DIRECTORY)

// The flag sits over the screenshot rather than beside it: the globe and the
// leaderboard are what make the card recognisable as the game, and a flag laid
// across them is what makes it recognisable as *yours*.
const FLAG_BOX_WIDTH = 480
const FLAG_BOX_HEIGHT = 340

// A white frame, so a flag with a white edge (Japan, Nepal's border) does not
// bleed into the darkened screenshot behind it.
const FLAG_FRAME = 6

const BACKGROUND_BRIGHTNESS = 0.5
const BACKGROUND_BLUR = 1.2

// 74 keeps a card around 20 kB. These are committed, 250-odd of them, so the
// quality is set where the flag is still clean and the repo does not grow by
// tens of megabytes.
const JPEG_QUALITY = 74

const background = await sharp(readFileSync(join(staticDir, "og-source.png")))
    .resize(CARD_WIDTH, CARD_HEIGHT, {
        fit: "contain",
        background: {r: 0, g: 0, b: 0, alpha: 1},
    })
    .modulate({brightness: BACKGROUND_BRIGHTNESS})
    .blur(BACKGROUND_BLUR)
    .flatten({background: {r: 0, g: 0, b: 0}})
    .toBuffer()

async function framedFlag(code: string): Promise<{buffer: Buffer, width: number, height: number}> {
    const source = join(staticDir, "countries", "svg", `${code}.svg`)

    // The SVGs have wildly different intrinsic sizes — Barbados declares itself
    // 24000 px wide — and sharp rasterises at `density` *before* resizing, so a
    // fixed density either blurs the small ones or blows past sharp's pixel
    // limit on the large ones. Read the declared size, which needs that limit
    // lifted for exactly that reason and rasterises nothing, then pick the
    // density that lands the raster just outside the box.
    const {width: intrinsicWidth = 0, height: intrinsicHeight = 0} =
        await sharp(source, {limitInputPixels: false}).metadata()
    const cover = Math.max(FLAG_BOX_WIDTH / intrinsicWidth, FLAG_BOX_HEIGHT / intrinsicHeight)
    const density = Math.min(2400, Math.max(1, Math.ceil(72 * cover)))

    const flag = await sharp(source, {density})
        .resize({width: FLAG_BOX_WIDTH, height: FLAG_BOX_HEIGHT, fit: "inside"})
        .png()
        .toBuffer()
    const {width = 0, height = 0} = await sharp(flag).metadata()

    const framed = await sharp({
        create: {
            width: width + 2 * FLAG_FRAME,
            height: height + 2 * FLAG_FRAME,
            channels: 4,
            background: {r: 255, g: 255, b: 255, alpha: 1},
        },
    })
        .composite([{input: flag, left: FLAG_FRAME, top: FLAG_FRAME}])
        .png()
        .toBuffer()

    return {buffer: framed, width: width + 2 * FLAG_FRAME, height: height + 2 * FLAG_FRAME}
}

// Rebuilt from scratch, so a country dropped from the list does not leave a
// card behind that nothing points at any more.
rmSync(cardDir, {recursive: true, force: true})
mkdirSync(cardDir, {recursive: true})

let bytes = 0

for (const [code] of ShareableCountries) {
    const flag = await framedFlag(code)

    const card = await sharp(background)
        .composite([{
            input: flag.buffer,
            left: Math.round((CARD_WIDTH - flag.width) / 2),
            top: Math.round((CARD_HEIGHT - flag.height) / 2),
        }])
        .jpeg({quality: JPEG_QUALITY, mozjpeg: true, progressive: false})
        .toBuffer()

    writeFileSync(join(cardDir, `${code}.jpg`), card)
    bytes += card.length
}

console.log(
    `wrote ${ShareableCountries.size} cards to ${CARD_DIRECTORY}/: ` +
    `${CARD_WIDTH}x${CARD_HEIGHT}, ${(bytes / 1024 / 1024).toFixed(1)} MB total, ` +
    `${(bytes / ShareableCountries.size / 1024).toFixed(0)} kB each`,
)
