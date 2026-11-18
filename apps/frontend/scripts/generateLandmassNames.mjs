import fs from "node:fs"
import path from "node:path"
import {boardDir, readBoard, readBorders, staticDir} from "./map/board.mjs"
import {lonLatOf} from "./map/lattice.mjs"
import {naturalEarth, NATURAL_EARTH_TAG} from "./map/naturalEarth.mjs"
import {evidenceOf} from "./map/names/evidence.mjs"
import {namesOf} from "./map/names/naming.mjs"
import {boardsAsked, writeBoardsTable, writePageFile} from "./writeBoard.ts"

const LAYERS = [
    "ne_50m_admin_0_countries",
    "ne_50m_admin_0_map_subunits",
    "ne_10m_admin_0_map_subunits",
    "ne_10m_geography_regions_polys",
    "ne_10m_geography_regions_points",
    "ne_10m_admin_1_states_provinces",
    "ne_10m_populated_places_simple",
]

const layers = {}
for (const name of LAYERS) layers[name] = await naturalEarth(name)
const countryNames = JSON.parse(fs.readFileSync(path.join(staticDir, "countries", "countries.json"), "utf8"))

// `npm run landmassNames -- earth-1` names one board's landmasses, and with no name every board's in /map.
for (const name of boardsAsked(process.argv.slice(2))) nameLandmasses(name)
writeBoardsTable()

// A landmass id is a row of a board's borders file: the names are the board's, keyed by its borders file.
function nameLandmasses(name) {
    const board = readBoard(boardDir(name))
    const borders = readBorders(board)
    if (borders.count !== board.count) {
        throw new Error(`${borders.name} has ${borders.count} tiles, ${board.file} has ${board.count}`)
    }

    const lon = new Float64Array(board.count)
    const lat = new Float64Array(board.count)
    for (let t = 0; t < board.count; t++) [lon[t], lat[t]] = lonLatOf(board.positions, t)

    const evidence = evidenceOf({count: board.count, lon, lat, codes: borders.codes, landmass: borders.landmass}, layers)
    const named = namesOf(evidence, countryNames)

    const lines = [...named]
        .filter(([, {main}]) => !main)
        .sort(([a], [b]) => a - b)
        .map(([id, {name: landmass}]) => `    ${JSON.stringify(String(id))}: ${JSON.stringify(landmass)}`)
    const bytes = Buffer.from(`{\n  "borders": ${JSON.stringify(borders.name)},\n  "names": {\n${lines.join(",\n")}\n  }\n}\n`)
    const fileName = writePageFile(name, "landmassNames", bytes)
    console.log(`named ${lines.length} landmasses of ${name} besides each country's own, Natural Earth ${NATURAL_EARTH_TAG}, `
        + `to static/boards/${name}/${fileName}`)
}
