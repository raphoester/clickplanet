import {readFileSync, writeFileSync} from "node:fs"
import {dirname, join, resolve} from "node:path"
import {fileURLToPath} from "node:url"
import sharp from "sharp"

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const staticDir = join(frontendRoot, "static")

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
