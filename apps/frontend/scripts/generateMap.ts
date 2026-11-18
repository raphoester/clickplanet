import {existsSync, readdirSync, writeFileSync} from "node:fs"

import {encodeRuns, runsOf} from "../src/domain/board.ts"
import {latticeIds, positionsHash} from "../src/domain/lattice.ts"
import {boardAsked, boardDir, readBoard, readBorders} from "./map/board.mjs"
import {encodeBorders} from "./map/bordersBinary.mjs"
import {capitalsOf} from "./map/capitals.mjs"
import {medianSpacingOf} from "./map/spacing.mjs"
import {SEA, loadGround} from "./map/ground.mjs"
import {landmasses} from "./map/landmasses.mjs"
import {keyOf, lattice, neighbours} from "./map/lattice.mjs"
import {NATURAL_EARTH_TAG, naturalEarth} from "./map/naturalEarth.mjs"
import {remap, remapLandmasses, remapSQL} from "./map/remap.mjs"
import {latticeGround} from "./map/oracle.mjs"
import {seatsOf} from "./map/seats.mjs"
import {syncBoards, writeBoardFile, writeBoardJSON, writeManifest} from "./writeBoard.ts"

// `npm run map:generate -- earth-2` writes one board of BOARD_DEFINITIONS; earth-1 when none is named.
const args = process.argv.slice(2)
const remapAt = args.includes("--remap") ? args[args.indexOf("--remap") + 1] : undefined
if (args.includes("--remap") && !remapAt) throw new Error("--remap needs a path to write the migration to")

const board = boardAsked(args.filter((arg) => arg !== remapAt))
const {cols} = board.lattice
// The board as /map holds it, which the remap follows ids from: none for a board written the first time.
const previous = existsSync(boardDir(board.name)) && readdirSync(boardDir(board.name)).length > 0 ? readBoard(boardDir(board.name)) : undefined
if (remapAt && previous === undefined) throw new Error(`--remap follows ${board.name}'s tiles from the board /map holds, and it holds none`)
// Read before the new borders file sweeps it away.
const previousBorders = previous === undefined ? undefined : readBorders(previous)
const ground = await loadGround()
const vertices = lattice(board.lattice)
const edges = neighbours(board.lattice)
const {at, to} = edges
const ids = latticeIds(board.lattice)
const {grounds: groundOfVertex, lakes, seated} = latticeGround(ground, vertices, edges, seatsOf(board.name))

// Keep the lattice's order (three's walk for earth-1, canonical for the rest): a tile's id is its rank among the tiles
// in it, the ledger holds earth-1's, and the remap migration relies on them staying monotonic.
const tilePositions: number[] = []
const tileIds: number[] = []
const grounds: string[] = []
const tileOf = new Int32Array(vertices.count).fill(-1)
for (let v = 0; v < vertices.count; v++) {
    const code = groundOfVertex[v]
    if (code === SEA) continue
    tileOf[v] = grounds.length
    grounds.push(code)
    tileIds.push(ids[v])
    tilePositions.push(vertices.positions[v * 3], vertices.positions[v * 3 + 1], vertices.positions[v * 3 + 2])
}
const count = grounds.length
const positions = Float32Array.from(tilePositions)
console.log(`Natural Earth ${NATURAL_EARTH_TAG}, ${board.name} at cols ${cols}${board.lattice.warp ? ", warped and rotated" : ""}: `
    + `${count} tiles of ${vertices.count} lattice points`)
console.log(`${lakes} lattice points on a country's ground are sea, because most of their cell is lake`)
const islands = seated.filter(({was}) => was === SEA).map(({code}) => code)
const onLand = seated.filter(({was}) => was !== SEA).map(({code, was}) => `${code} from ${was}`)
console.log(`seated ${seated.length} countries the lattice misses: ${islands.length} on a tile of their own at sea`
    + ` (${islands.join(" ")}), ${onLand.length} on a neighbour's (${onLand.join(", ")})`)

const pieces = landmasses({count, positions, grounds, tileOf, at, to})
const countries = new Set(pieces.codes.slice(1))
console.log(`${pieces.codes.length - 1} landmasses across ${countries.size} countries`)

const runs = runsOf(tileIds)
const runsFile = writeBoardFile(board.name, "runs", new Uint8Array(encodeRuns(runs)))
console.log(`wrote map/boards/${board.name}/${runsFile}: ${count} tiles in ${runs.length} runs`)

const borders = encodeBorders(pieces)
const bordersFile = writeBoardFile(board.name, "borders", borders.bytes)
console.log(`wrote map/boards/${board.name}/${bordersFile}: ${borders.bytes.byteLength} bytes`)

const [places, countryShapes] = await Promise.all([
    naturalEarth("ne_10m_populated_places"),
    naturalEarth("ne_50m_admin_0_countries"),
])
const capitals = capitalsOf(places, countryShapes, {
    codes: pieces.codes,
    landmass: pieces.assignment,
    positions,
    frames: pieces.frames,
    totals: pieces.totals,
})
const capitalsFile = writeBoardJSON(board.name, "capitals", {tiles: count, capitals})
console.log(`wrote map/boards/${board.name}/${capitalsFile.fileName}: ${Object.keys(capitals).length} capitals`)

const manifest = writeManifest(board.name, {
    body: board.body,
    lattice: board.lattice,
    tiles: count,
    spacing: medianSpacingOf(positions, {at, to, tileOf}),
    positionsHash: positionsHash(positions),
})
console.log(`wrote map/boards/${board.name}/${manifest}`)
syncBoards()

if (previous !== undefined) {
    const before: string[] = []
    for (let t = 0; t < previous.count; t++) {
        before.push(keyOf(previous.positions[t * 3], previous.positions[t * 3 + 1], previous.positions[t * 3 + 2]))
    }
    const after: string[] = []
    for (let t = 0; t < count; t++) after.push(keyOf(positions[t * 3], positions[t * 3 + 1], positions[t * 3 + 2]))
    const moved = remap(before, after)
    console.log(
        `ids: ${moved.kept} kept, ${moved.added} added, ${moved.removed.length} gone`
        + ` — ${moved.runs.length} runs`,
    )

    if (remapAt && previousBorders !== undefined) {
        if (!remapAt.endsWith(".up.sql")) throw new Error("--remap wants the .up.sql path; the .down.sql is written beside it")
        const plan = {
            ...moved,
            before: previous.count,
            after: count,
            from: previous.manifest.files.runs,
            to: runsFile,
            landmasses: remapLandmasses(previousBorders, {codes: pieces.codes, landmass: pieces.assignment}, moved.runs),
            borders: {from: previousBorders.name, to: bordersFile},
        }
        console.log(`landmasses: ${plan.landmasses.length} of ${previousBorders.codes.length - 1} followed to the new board`)
        const downAt = `${remapAt.slice(0, -".up.sql".length)}.down.sql`
        writeFileSync(remapAt, remapSQL(plan))
        writeFileSync(downAt, remapSQL(plan, {down: true}))
        console.log(`wrote ${remapAt} and ${downAt}`)
    }
}

console.log(`
next:
  npm run bridges -- ${board.name}       # the crossings, from the new tiles and capitals
  npm run terrain -- ${board.name}       # each lattice point's look, from the new tiles
  cd ../backend && make map        # copy the board, then commit every copy
  cd ../backoffice && npm run map  # copy the manifest and the runs
  npm run borderLines -- ${board.name}   # traced from the board, stale until it is run
  npm run landmassNames -- ${board.name} # each landmass's name, keyed by the new borders file
  npm run map:audit -- ${board.name}     # every fault should read zero
  a new board's row in planet's boards: a migration with its name and positionsHash in 8 hex`)
