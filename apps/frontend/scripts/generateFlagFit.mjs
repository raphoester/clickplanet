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

const fit = {}
for (const [code, region] of Object.entries(atlas)) {
    if (region.width <= INSET * 2 || region.height <= INSET * 2) continue
    // Bands running across the flag means every line the other way is uniform.
    const horizontalBands = uniformLines(region, false)
    const verticalBands = uniformLines(region, true)
    if (horizontalBands || verticalBands) fit[code] = true
}

fs.writeFileSync("static/countries/flagFit.json", JSON.stringify(fit, null, 0) + "\n")
console.log(`${Object.keys(fit).length} of ${Object.keys(atlas).length} flags are plain bands and may be stretched`)
for (const code of ["fr", "ru", "it", "de", "nl", "be", "pl", "ps", "es", "cn", "gb", "us", "jp", "pt", "lk", "il", "sd", "dz"]) {
    console.log("  " + code.padEnd(4), fit[code] ? "stretch" : "keep shape")
}
