import fs from "node:fs"
import path from "node:path"
import {readBorders, readCoordinates, staticDir} from "./map/blob.mjs"
import {naturalEarth, NATURAL_EARTH_TAG} from "./map/naturalEarth.mjs"
import {evidenceOf} from "./map/names/evidence.mjs"
import {namesOf} from "./map/names/naming.mjs"

const LAYERS = [
    "ne_50m_admin_0_countries",
    "ne_50m_admin_0_map_subunits",
    "ne_10m_admin_0_map_subunits",
    "ne_10m_geography_regions_polys",
    "ne_10m_geography_regions_points",
    "ne_10m_admin_1_states_provinces",
    "ne_10m_populated_places_simple",
]

const coordinates = readCoordinates()
const borders = readBorders()
if (borders.count !== coordinates.count) {
    throw new Error(`${borders.name} has ${borders.count} tiles, ${coordinates.name} has ${coordinates.count}`)
}

const lon = new Float64Array(coordinates.count)
const lat = new Float64Array(coordinates.count)
for (let t = 0; t < coordinates.count; t++) {
    lon[t] = (coordinates.uvs[t * 2] - 0.5) * 360
    lat[t] = (coordinates.uvs[t * 2 + 1] - 0.5) * 180
}

const layers = {}
for (const name of LAYERS) layers[name] = await naturalEarth(name)

const countryNames = JSON.parse(fs.readFileSync(path.join(staticDir, "countries", "countries.json"), "utf8"))
const evidence = evidenceOf({count: coordinates.count, lon, lat, codes: borders.codes, landmass: borders.landmass}, layers)
const named = namesOf(evidence, countryNames)

const lines = [...named]
    .filter(([, {main}]) => !main)
    .sort(([a], [b]) => a - b)
    .map(([id, {name}]) => `    ${JSON.stringify(String(id))}: ${JSON.stringify(name)}`)
const out = path.join(staticDir, "landmassNames.json")
fs.writeFileSync(out, `{\n  "borders": ${JSON.stringify(borders.name)},\n  "names": {\n${lines.join(",\n")}\n  }\n}\n`)
console.log(`named ${lines.length} landmasses besides each country's own, Natural Earth ${NATURAL_EARTH_TAG}, to ${out}`)
