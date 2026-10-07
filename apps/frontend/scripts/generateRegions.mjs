import fs from "node:fs"
import path from "node:path"
import {fileURLToPath} from "node:url"
import {naturalEarth, NATURAL_EARTH_TAG} from "./map/naturalEarth.mjs"
import {countryCodeOf} from "./map/ground.mjs"

const out = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "static", "countries", "regions.json")

const countries = await naturalEarth("ne_50m_admin_0_countries")

// Natural Earth puts the islands far from any coast in no continent.
const OPEN_OCEAN = "Seven seas (open ocean)"

const regions = {}
for (const feature of countries.features) {
    const code = countryCodeOf(feature)
    const continent = String(feature.properties.CONTINENT)
    if (code === null || regions[code] || continent === OPEN_OCEAN) continue
    regions[code] = continent
}

const lines = Object.entries(regions)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([code, region]) => `  ${JSON.stringify(code)}: ${JSON.stringify(region)}`)
fs.writeFileSync(out, `{\n${lines.join(",\n")}\n}\n`)
console.log(`wrote the continent of ${lines.length} countries, Natural Earth ${NATURAL_EARTH_TAG}, to ${out}`)
