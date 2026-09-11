import {readFileSync, writeFileSync} from "node:fs"
import {dirname, join, resolve} from "node:path"
import {fileURLToPath} from "node:url"
import sharp from "sharp"
import {CARD_HEIGHT, CARD_WIDTH} from "../src/domain/shareCard.ts"

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const staticDir = join(frontendRoot, "static")

const source = readFileSync(join(staticDir, "og-source.png"))

const image = await sharp(source)
    .resize(CARD_WIDTH, CARD_HEIGHT, {
        fit: "contain",
        background: {r: 0, g: 0, b: 0, alpha: 1},
    })
    .flatten({background: {r: 0, g: 0, b: 0}})
    .jpeg({quality: 86, mozjpeg: true, progressive: false})
    .toBuffer()

writeFileSync(join(staticDir, "og-image.jpg"), image)

console.log(
    `wrote static/og-image.jpg: ${CARD_WIDTH}x${CARD_HEIGHT}, ` +
    `${(image.length / 1024).toFixed(0)} kB (from ${(source.length / 1024).toFixed(0)} kB)`,
)
