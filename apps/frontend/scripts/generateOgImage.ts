// Rewrites the link preview from the screenshot it is built from.
//
//   npm run og-image
//
// The source is `static/og-source.png`, kept in the repo and not deployed. The output is
// `static/og-image-<hash>.jpg`, and the og:image / twitter:image URLs in index.html and play.html
// are rewritten to name it. It is content-addressed because scrapers cache a preview by its URL:
// under the same name, a link shared after the change still showed the old picture.
import {createHash} from "node:crypto"
import {readdirSync, readFileSync, unlinkSync, writeFileSync} from "node:fs"
import {dirname, join, resolve} from "node:path"
import {fileURLToPath} from "node:url"
import sharp from "sharp"

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const staticDir = join(frontendRoot, "static")
const pages = ["index.html", "play.html"].map(page => join(frontendRoot, page))

export const OG_IMAGE_WIDTH = 1200
export const OG_IMAGE_HEIGHT = 627

const NAME = /^og-image(-[0-9a-f]{8})?\.jpg$/
const IMAGE_URL = /\/static\/og-image(-[0-9a-f]{8})?\.jpg/g

const source = readFileSync(join(staticDir, "og-source.png"))

const image = await sharp(source)
    .resize(OG_IMAGE_WIDTH, OG_IMAGE_HEIGHT, {
        fit: "contain",
        background: {r: 0, g: 0, b: 0, alpha: 1},
    })
    .flatten({background: {r: 0, g: 0, b: 0}})
    .jpeg({quality: 86, mozjpeg: true, progressive: false})
    .toBuffer()

const fileName = `og-image-${createHash("sha256").update(image).digest("hex").slice(0, 8)}.jpg`
for (const entry of readdirSync(staticDir)) {
    if (NAME.test(entry) && entry !== fileName) unlinkSync(join(staticDir, entry))
}
writeFileSync(join(staticDir, fileName), image)

for (const page of pages) {
    const html = readFileSync(page, "utf8")
    if (html.search(IMAGE_URL) === -1) throw new Error(`${page} names no /static/og-image*.jpg to rewrite`)
    writeFileSync(page, html.replaceAll(IMAGE_URL, `/static/${fileName}`))
}

console.log(
    `wrote static/${fileName}: ${OG_IMAGE_WIDTH}x${OG_IMAGE_HEIGHT}, ` +
    `${(image.length / 1024).toFixed(0)} kB (from ${(source.length / 1024).toFixed(0)} kB)`,
)
