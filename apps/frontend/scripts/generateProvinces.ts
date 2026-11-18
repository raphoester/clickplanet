import {bordersHashOf, encodeProvinces} from "../src/domain/provinces.ts"
import {BOARD_DEFINITIONS} from "./map/board.mjs"
import {NATURAL_EARTH_TAG} from "./map/naturalEarth.mjs"
import {cutOf, landmassNamesOf, overridesOf, provinceNamesOf} from "./map/provinces.mjs"
import {syncBoards, writeBoardFile, writeManifest, writePageFile} from "./writeBoard.ts"

// `npm run provinces -- earth-2` cuts one board's provinces, and with no name every board that has a floor.
const asked = process.argv.slice(2).filter((arg) => !arg.startsWith("-"))
const boards = asked.length > 0 ? asked : Object.values(BOARD_DEFINITIONS).filter((board) => board.provinceFloor).map((board) => board.name)
for (const name of boards) {
    const floor = BOARD_DEFINITIONS[name]?.provinceFloor
    if (floor === undefined) throw new Error(`${name} has no provinces: give it a provinceFloor in BOARD_DEFINITIONS`)
    await cutBoard(name, floor)
}
syncBoards()

async function cutBoard(name: string, floor: number) {
    const {board, borders, units, under, cut} = await cutOf(name, floor)
    const {names, alike, overridden, regions, largest, wholes, countries, renamed} = provinceNamesOf(cut, units, overridesOf(name), {
        codes: borders.codes,
        landmassNames: landmassNamesOf(name, borders),
    })
    if (alike.length > 0) throw new Error(`${name}: provinces an override must name: ${alike.join("; ")}`)

    const bytes = encodeProvinces({
        borders: bordersHashOf(borders.name),
        province: cut.province,
        landmass: cut.landmasses,
        frames: cut.frames,
        tiles: cut.totals,
    })
    const file = writeBoardFile(name, "provinces", bytes)
    const manifest = writeManifest(name)

    const lines = [...names].sort(([a], [b]) => a - b).map(([k, province]) => `    ${JSON.stringify(String(k))}: ${JSON.stringify(province)}`)
    const namesFile = writePageFile(name, "provinceNames", Buffer.from(`{\n  "provinces": ${JSON.stringify(file)},\n  "names": {\n${lines.join(",\n")}\n  }\n}\n`))

    const count = cut.totals.length - 1
    let big = 0
    for (let k = 1; k <= count; k++) if (cut.totals[k] >= floor) big++
    console.log(`${name}, Natural Earth ${NATURAL_EARTH_TAG} ${board.count} tiles: ${under.foreign} under another country's unit, ${under.none} under none, `
        + `${cut.stats.filled} given their landmass's nearest unit; ${cut.stats.pieces} pieces, ${cut.stats.merges} joined under ${floor} tiles
  ${count} provinces, ${big} of ${floor} tiles or more
  names: ${overridden} given by hand, ${wholes} a whole landmass's (${renamed} numbered apart from a province of that name),
    ${regions} a whole region's, ${largest} their largest unit's, the rest their one unit's; ${countries} a country's whole
    main landmass, which reads as the country
wrote map/boards/${name}/${file} (${(bytes.byteLength / 1024).toFixed(0)} KB) and ${manifest}, and static/boards/${name}/${namesFile}`)
}

console.log(`next:
  cd ../backend && make map   # copy the boards, then commit every copy
  cd ../backoffice && npm run map   # copy the manifests
  npm run map:audit -- <board>      # every province on one landmass, of the floor or more
  then Claude Code's translate skill (.claude/skills/translate): the province names in every language`)
