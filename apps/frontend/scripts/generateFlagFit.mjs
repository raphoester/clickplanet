// Which flags survive being stretched to the shape of a country.
//
// A flag painted across a landmass has to fit a shape nobody chose. Keeping its
// proportions and cropping works for a squarish country and fails badly for a
// narrow one: Portugal and Sri Lanka are tall and thin, so a French tricolour
// fitted over them shows nothing but the white band down the middle.
//
// Stretching is fine for a flag that is only bands — France squeezed narrow is
// still blue, white and red — and wrong for one carrying a device, which
// stretching deforms. That is a property of the artwork, so it is read off the
// artwork: a flag is stretchable when every row of it is one colour, or every
// column is.
//
// It also records where each flag carries its identity. A flag painted on a
// narrow country is cropped to a vertical slice of itself, and taking that slice
// from the middle throws away exactly what names it: Palestine on Argentina came
// out green, white and black with no red, because the red is the hoist triangle.
// The columns that least resemble the flag's average column are where its
// content is — the triangle for Palestine, the canton for the USA, the disc for
// Japan, and nowhere in particular for a plain tricolour. Their centre of mass
// is the point a crop should be taken around.
//
//   node scripts/generateFlagFit.mjs
import fs from "node:fs"
const sharp = (await import(process.cwd() + "/node_modules/sharp/lib/index.js")).default

const atlas = JSON.parse(fs.readFileSync("static/countries/atlas.json", "utf8"))
const {ATLAS_URL} = fs.readFileSync("src/app/viewer/atlasAsset.ts", "utf8")
    .match(/"(?<ATLAS_URL>[^"]+)"/).groups
const {data, info} = await sharp("." + ATLAS_URL).raw().toBuffer({resolveWithObject: true})
const {width, channels} = info

// Room for the artwork's own antialiasing along a band edge, and for the thin
// outline some flags are drawn with.
const TOLERANCE = 14
const INSET = 2

const at = (x, y) => {
    const i = (y * width + x) * channels
    return [data[i], data[i + 1], data[i + 2]]
}

// Every line along `axis` is a single colour.
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

// Where along `axis` the flag stops looking like itself.
function focusOf(region, vertical) {
    const outer = vertical ? region.width : region.height
    const inner = vertical ? region.height : region.width

    // Every line, and the average of them all.
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

    // A flag of plain bands has no line that stands out, so nothing to aim at.
    return total < 1e-6 ? 0.5 : weighted / total / outer
}

const fit = {}
for (const [code, region] of Object.entries(atlas)) {
    if (region.width <= INSET * 2 || region.height <= INSET * 2) continue
    // Bands running across the flag means every line the other way is uniform.
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
