import fs from "node:fs"
const sharp = (await import(process.cwd() + "/node_modules/sharp/lib/index.js")).default

const atlas = JSON.parse(fs.readFileSync("static/countries/atlas.json", "utf8"))
const {ATLAS_URL} = fs.readFileSync("src/app/viewer/atlasAsset.ts", "utf8")
    .match(/"(?<ATLAS_URL>[^"]+)"/).groups
const {data, info} = await sharp("." + ATLAS_URL).raw().toBuffer({resolveWithObject: true})
const {width, channels} = info

const TOLERANCE = 14
const INSET = 2

const at = (x, y) => {
    const i = (y * width + x) * channels
    return [data[i], data[i + 1], data[i + 2]]
}

function uniformLines(region, vertical) {
    const outer = vertical ? region.width : region.height
    const inner = vertical ? region.height : region.width
    for (let a = INSET; a < outer - INSET; a++) {
        let lo = [255, 255, 255], hi = [0, 0, 0]
        for (let b = INSET; b < inner - INSET; b++) {
            const [x, y] = vertical
                ? [region.x + a, region.y + b]
                : [region.x + b, region.y + a]
            const pixel = at(x, y)
            for (let c = 0; c < 3; c++) {
                lo[c] = Math.min(lo[c], pixel[c])
                hi[c] = Math.max(hi[c], pixel[c])
            }
        }
        for (let c = 0; c < 3; c++) if (hi[c] - lo[c] > TOLERANCE) return false
    }
    return true
}

function focusOf(region, vertical) {
    const outer = vertical ? region.width : region.height
    const inner = vertical ? region.height : region.width

    const lines = []
    const mean = new Float64Array(inner * 3)
    for (let a = 0; a < outer; a++) {
        const line = new Float64Array(inner * 3)
        for (let b = 0; b < inner; b++) {
            const [x, y] = vertical ? [region.x + a, region.y + b] : [region.x + b, region.y + a]
            const pixel = at(x, y)
            for (let c = 0; c < 3; c++) {
                line[b * 3 + c] = pixel[c]
                mean[b * 3 + c] += pixel[c] / outer
            }
        }
        lines.push(line)
    }

    let total = 0
    let weighted = 0
    for (let a = 0; a < outer; a++) {
        let apart = 0
        for (let i = 0; i < inner * 3; i++) apart += Math.abs(lines[a][i] - mean[i])
        apart /= inner
        total += apart
        weighted += apart * (a + 0.5)
    }

    return total < 1e-6 ? 0.5 : weighted / total / outer
}

// The flag drawn this wide from its 100px sprite: the mean difference to itself drawn from its SVG.
const DETAIL_WIDTH = 400
const SPRITE_WIDTH = 100

async function lossOf(code) {
    const path = `static/countries/svg/${code}.svg`
    if (!fs.existsSync(path)) return undefined
    const svg = fs.readFileSync(path)
    const {width: svgWidth} = await sharp(svg, {limitInputPixels: false}).metadata()
    const density = Math.max(1, 72 * 2 * DETAIL_WIDTH / svgWidth)
    const crisp = await sharp(svg, {density, limitInputPixels: false})
        .resize({width: DETAIL_WIDTH}).flatten({background: "#ffffff"}).removeAlpha()
        .raw().toBuffer({resolveWithObject: true})
    const {width, height} = crisp.info
    const sprite = await sharp(crisp.data, {raw: {width, height, channels: 3}})
        .resize({width: SPRITE_WIDTH, height: Math.round(height * SPRITE_WIDTH / width), kernel: "lanczos3"})
        .raw().toBuffer({resolveWithObject: true})
    const blurred = await sharp(sprite.data, {raw: sprite.info}).resize({width, height, kernel: "linear"}).raw().toBuffer()
    let sum = 0
    for (let i = 0; i < blurred.length; i++) sum += Math.abs(blurred[i] - crisp.data[i])
    return sum / blurred.length / 255
}

const fit = {}
const losses = {}
for (const [code, region] of Object.entries(atlas)) {
    if (region.width <= INSET * 2 || region.height <= INSET * 2) continue
    const horizontalBands = uniformLines(region, false)
    const verticalBands = uniformLines(region, true)
    fit[code] = {
        stretch: horizontalBands || verticalBands,
        focus: [
            Number(focusOf(region, true).toFixed(4)),
            Number(focusOf(region, false).toFixed(4)),
        ],
    }
    const loss = await lossOf(code)
    if (loss !== undefined) losses[code] = loss
}

// A flag with no SVG is never drawn in a slot, and has no detail.
const sorted = Object.values(losses).sort((a, b) => a - b)
const median = sorted[Math.floor(sorted.length / 2)]
for (const [code, loss] of Object.entries(losses)) fit[code].detail = Number((loss / median).toFixed(2))

fs.writeFileSync("static/countries/flagFit.json", JSON.stringify(fit, null, 0) + "\n")
const stretchy = Object.values(fit).filter((f) => f.stretch).length
console.log(`${stretchy} of ${Object.keys(fit).length} flags are plain bands and may be stretched`)
console.log("code  stretch  focus across / down  detail")
for (const code of ["fr", "ru", "ps", "sd", "us", "jp", "ca", "br", "cn", "gb", "es", "cl", "in", "ch", "pl", "sa", "ir", "mx"]) {
    const f = fit[code]
    if (f) console.log("  " + code.padEnd(4), (f.stretch ? "yes" : "no ").padEnd(8), f.focus[0].toFixed(2), f.focus[1].toFixed(2), "       ", f.detail?.toFixed(2) ?? "-")
}
