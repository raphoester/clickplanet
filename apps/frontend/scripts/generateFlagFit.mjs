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

const fit = {}
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
}

fs.writeFileSync("static/countries/flagFit.json", JSON.stringify(fit, null, 0) + "\n")
const stretchy = Object.values(fit).filter((f) => f.stretch).length
console.log(`${stretchy} of ${Object.keys(fit).length} flags are plain bands and may be stretched`)
console.log("code  stretch  focus across / down")
for (const code of ["fr", "ru", "ps", "sd", "us", "jp", "ca", "br", "cn", "gb", "es", "cl", "in", "ch"]) {
    const f = fit[code]
    if (f) console.log("  " + code.padEnd(4), (f.stretch ? "yes" : "no ").padEnd(8), f.focus[0].toFixed(2), f.focus[1].toFixed(2))
}
