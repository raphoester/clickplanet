import fs from "node:fs"
import path from "node:path"
import {fileURLToPath} from "node:url"
import {naturalEarth, NATURAL_EARTH_TAG} from "./map/naturalEarth.mjs"
import {countryCodeOf} from "./map/ground.mjs"

const out = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "static", "countries", "regions.json")

const countries = await naturalEarth("ne_50m_admin_0_countries")

// Natural Earth puts the islands far from any coast in no continent.
const OPEN_OCEAN = "Seven seas (open ocean)"

// Asia is too big to be one place: nobody calls Saudi Arabia "Asia". Its parts are named as people name them, the
// Middle East being western Asia and Iran, which the World Bank counts in it.
const MIDDLE_EAST = "Middle East"
const ASIA = {
    "Western Asia": MIDDLE_EAST,
    "Southern Asia": "South Asia",
    "Eastern Asia": "East Asia",
    "South-Eastern Asia": "Southeast Asia",
    "Central Asia": "Central Asia",
}

function regionOf({CONTINENT, SUBREGION, REGION_WB}) {
    if (CONTINENT !== "Asia") return String(CONTINENT)
    return REGION_WB === "Middle East & North Africa" ? MIDDLE_EAST : ASIA[SUBREGION]
}

const regions = {}
for (const feature of countries.features) {
    const code = countryCodeOf(feature)
    const region = regionOf(feature.properties)
    if (code === null || regions[code] || region === undefined || region === OPEN_OCEAN) continue
    regions[code] = region
}

const lines = Object.entries(regions)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([code, region]) => `  ${JSON.stringify(code)}: ${JSON.stringify(region)}`)
fs.writeFileSync(out, `{\n${lines.join(",\n")}\n}\n`)
console.log(`wrote the region of ${lines.length} countries, Natural Earth ${NATURAL_EARTH_TAG}, to ${out}`)
