import {createHash} from "node:crypto"
import {mkdirSync, readdirSync, rmSync, writeFileSync} from "node:fs"
import {dirname, join, resolve} from "node:path"
import {fileURLToPath} from "node:url"

import {type Board, encodeRuns, runsOf} from "../../src/domain/board.ts"
import {latticeIds, positionsHash} from "../../src/domain/lattice.ts"
import {LAND, encodeTerrain, seaKind} from "../../src/domain/terrain.ts"
import {readBoard, readBorders, readCapitals, readTerrain} from "./board.mjs"
import {encodeBorders} from "./bordersBinary.mjs"
import {SEA} from "./ground.mjs"
import {landmasses} from "./landmasses.mjs"
import {lattice, neighbours} from "./lattice.mjs"
import {medianSpacingOf} from "./spacing.mjs"

// The tests' second board (`npm run map:fixture`): a coarse earth cut from earth-1, at `cols` 31 rather than 301, so
// its tiles are about ten times as far apart and a hundredth as dense. It ships in each app's test data alone, never
// in /map/boards, generated/ or static/: no app plays it, and planet keeps no row of it but in a test's database.
//
// A point of its lattice is a tile when an earth-1 tile is within EARTH_REACH of earth-1's spacing of it, on that
// tile's country, with that tile's kind of land. Its capitals are each country's tile nearest earth-1's capital, and it
// has no bridges.
const NAME = "fixture-1"
const COLS = 31
const EARTH_REACH = 1.5

const appsDir = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..", "..")
const COPIES: {dir: string, kinds: readonly Kind[]}[] = [
    {dir: join(appsDir, "backend", "internal", "shared", "cpboards", "testdata", NAME), kinds: ["runs", "borders", "capitals", "bridges", "terrain"]},
    {dir: join(appsDir, "frontend", "src", "testdata", "boards", NAME), kinds: ["runs", "borders", "capitals", "bridges", "terrain"]},
    {dir: join(appsDir, "backoffice", "src", "testdata", "boards", NAME), kinds: ["runs"]},
]

type Kind = "runs" | "borders" | "capitals" | "bridges" | "terrain"
const EXTENSIONS: Record<Kind, string> = {runs: "bin", borders: "bin", capitals: "json", bridges: "json", terrain: "bin"}

const earth = readBoard()
const earthBorders = readBorders(earth)
const earthCapitals = readCapitals(earth)
const earthTerrain = readTerrain(earth).terrain

const vertices = lattice(COLS)
const edges = neighbours(COLS)
const ids = latticeIds(COLS)
const reach = Math.cos(EARTH_REACH * earth.manifest.spacing)

// Each lattice point's nearest earth-1 tile, when it is close enough to be land.
const nearest = new Int32Array(vertices.count).fill(-1)
for (let v = 0; v < vertices.count; v++) {
    const x = vertices.positions[v * 3], y = vertices.positions[v * 3 + 1], z = vertices.positions[v * 3 + 2]
    let best = reach
    for (let t = 0; t < earth.count; t++) {
        const dot = x * earth.positions[t * 3] + y * earth.positions[t * 3 + 1] + z * earth.positions[t * 3 + 2]
        if (dot > best) {
            best = dot
            nearest[v] = t
        }
    }
}

const grounds: string[] = []
const tileIds: number[] = []
const tilePositions: number[] = []
const earthTileOf: number[] = []
const tileOf = new Int32Array(vertices.count).fill(-1)
for (let v = 0; v < vertices.count; v++) {
    const from = nearest[v]
    if (from < 0) continue
    tileOf[v] = grounds.length
    grounds.push(earthBorders.codes[earthBorders.landmass[from]])
    tileIds.push(ids[v])
    earthTileOf.push(from)
    tilePositions.push(vertices.positions[v * 3], vertices.positions[v * 3 + 1], vertices.positions[v * 3 + 2])
}
const count = grounds.length
const positions = Float32Array.from(tilePositions)

const pieces = landmasses({count, positions, grounds, tileOf, at: edges.at, to: edges.to})

const capitals: Record<string, number> = {}
for (const [country, capital] of Object.entries(earthCapitals.capitals)) {
    const t = capital - 1
    let best = -2
    for (let tile = 0; tile < count; tile++) {
        if (grounds[tile] !== country) continue
        const dot = positions[tile * 3] * earth.positions[t * 3] + positions[tile * 3 + 1] * earth.positions[t * 3 + 1]
            + positions[tile * 3 + 2] * earth.positions[t * 3 + 2]
        if (dot > best) {
            best = dot
            capitals[country] = tile + 1
        }
    }
}

// Land takes its earth-1 tile's kind; the sea its steps from land, as `npm run terrain` gives it.
const kinds = new Uint8Array(vertices.count).fill(seaKind(Number.MAX_SAFE_INTEGER))
const steps = new Int32Array(vertices.count).fill(-1)
let front: number[] = []
for (let v = 0; v < vertices.count; v++) {
    if (tileOf[v] < 0) continue
    const kind = earthTerrain[earth.ids[nearest[v]]]
    kinds[ids[v]] = kind < LAND.length ? kind : LAND.indexOf("grass")
    steps[v] = 0
    front.push(v)
}
for (let step = 1; front.length > 0; step++) {
    const next: number[] = []
    for (const v of front) {
        for (let e = edges.at[v]; e < edges.at[v + 1]; e++) {
            const other = edges.to[e]
            if (steps[other] >= 0) continue
            steps[other] = step
            kinds[ids[other]] = seaKind(step)
            next.push(other)
        }
    }
    front = next
}

const files: Record<Kind, Uint8Array> = {
    runs: new Uint8Array(encodeRuns(runsOf(tileIds))),
    borders: encodeBorders(pieces).bytes,
    capitals: Buffer.from(`${JSON.stringify({tiles: count, capitals})}\n`),
    bridges: Buffer.from(`${JSON.stringify({tiles: count, bridges: []})}\n`),
    terrain: encodeTerrain(kinds),
}
const names = Object.fromEntries(Object.entries(files)
    .map(([kind, bytes]) => [kind, `${kind}-${hashOf(bytes)}.${EXTENSIONS[kind as Kind]}`])) as Record<Kind, string>

const manifest: Board = {
    name: NAME,
    body: "earth",
    lattice: {cols: COLS, warp: null, corners: null, order: "three"},
    tiles: count,
    spacing: medianSpacingOf(positions, {at: edges.at, to: edges.to, tileOf}),
    positionsHash: positionsHash(positions),
    files: names,
}
if (manifest.positionsHash === earth.manifest.positionsHash) throw new Error(`${NAME} hashes as earth-1 does`)
const manifestBytes = Buffer.from(`${JSON.stringify(manifest, null, 2)}\n`)
const manifestName = `board-${hashOf(manifestBytes)}.json`

for (const {dir, kinds: carried} of COPIES) {
    rmSync(dir, {recursive: true, force: true})
    mkdirSync(dir, {recursive: true})
    writeFileSync(join(dir, manifestName), manifestBytes)
    for (const kind of carried) writeFileSync(join(dir, names[kind]), files[kind])
    console.log(`wrote ${dir}: ${readdirSync(dir).join(", ")}`)
}
console.log(`${NAME}: ${count} tiles of ${vertices.count} points, ${pieces.codes.length - 1} landmasses, `
    + `${Object.keys(capitals).length} capitals, spacing ${manifest.spacing} (earth-1's × ${(manifest.spacing / earth.manifest.spacing).toFixed(3)}), `
    + `positions hash ${manifest.positionsHash.toString(16).padStart(8, "0")}`)

function hashOf(bytes: Uint8Array): string {
    return createHash("sha256").update(bytes).digest("hex").slice(0, 8)
}

// SEA is the oracle's word for no country; no fixture tile is one.
if (grounds.includes(SEA)) throw new Error("a fixture tile is on no country")
