import {writeFileSync} from "node:fs"

import {encodeBorders} from "./map/bordersBinary.mjs"
import {SEA, loadGround} from "./map/ground.mjs"
import {landmasses} from "./map/landmasses.mjs"
import {DETAIL, keyOf, lattice, lonLatOf, neighbours} from "./map/lattice.mjs"
import {NATURAL_EARTH_TAG} from "./map/naturalEarth.mjs"
import {readCoordinates} from "./map/blob.mjs"
import {remap, remapSQL} from "./map/remap.mjs"
import {bordersPath, staticBordersPath, writeBordersAsset, writeCoordinatesBinary} from "./writeCoordinates.ts"

const args = process.argv.slice(2)
const remapAt = args.includes("--remap") ? args[args.indexOf("--remap") + 1] : undefined
if (args.includes("--remap") && !remapAt) throw new Error("--remap needs a path to write the migration to")

const previous = readCoordinates()
const ground = await loadGround()
const {count: vertices, positions, uvs} = lattice(DETAIL)

// Keep lattice order: the remap migration relies on ids staying monotonic.
const tilePositions: number[] = []
const tileUVs: number[] = []
const grounds: string[] = []
const tileOf = new Int32Array(vertices).fill(-1)
for (let v = 0; v < vertices; v++) {
    const code = ground.groundOf(...lonLatOf(uvs[v * 2], uvs[v * 2 + 1]))
    if (code === SEA) continue
    tileOf[v] = grounds.length
    grounds.push(code)
    tilePositions.push(positions[v * 3], positions[v * 3 + 1], positions[v * 3 + 2])
    tileUVs.push(uvs[v * 2], uvs[v * 2 + 1])
}
const count = grounds.length
console.log(`Natural Earth ${NATURAL_EARTH_TAG}, detail ${DETAIL}: ${count} tiles of ${vertices} lattice vertices`)

const {at, to} = neighbours(DETAIL)
const pieces = landmasses({
    count,
    positions: Float32Array.from(tilePositions),
    grounds,
    tileOf,
    at,
    to,
})
const countries = new Set(pieces.codes.slice(1))
console.log(`${pieces.codes.length - 1} landmasses across ${countries.size} countries`)

const coordinates = writeCoordinatesBinary({
    positions: tilePositions,
    uvs: tileUVs,
    length: count,
})
console.log(`wrote map/${coordinates.fileName}: ${coordinates.tiles} tiles, ${coordinates.byteLength} bytes`)

const borders = encodeBorders(pieces)
for (const path of [bordersPath(borders.fileName), staticBordersPath(borders.fileName)]) {
    writeFileSync(path, borders.bytes)
}
writeBordersAsset(borders.fileName)
console.log(`wrote map/${borders.fileName}: ${borders.bytes.byteLength} bytes`)

const before: string[] = []
for (let t = 0; t < previous.count; t++) {
    before.push(keyOf(previous.positions[t * 3], previous.positions[t * 3 + 1], previous.positions[t * 3 + 2]))
}
const after: string[] = []
for (let t = 0; t < count; t++) {
    after.push(keyOf(tilePositions[t * 3], tilePositions[t * 3 + 1], tilePositions[t * 3 + 2]))
}
const moved = remap(before, after)
console.log(
    `ids: ${moved.kept} kept, ${moved.added} added, ${moved.removed.length} gone`
    + ` — ${moved.runs.length} runs`,
)

if (remapAt) {
    if (!remapAt.endsWith(".up.sql")) throw new Error("--remap wants the .up.sql path; the .down.sql is written beside it")
    const plan = {
        ...moved,
        before: previous.count,
        after: count,
        from: previous.name,
        to: coordinates.fileName,
    }
    const downAt = `${remapAt.slice(0, -".up.sql".length)}.down.sql`
    writeFileSync(remapAt, remapSQL(plan))
    writeFileSync(downAt, remapSQL(plan, {down: true}))
    console.log(`wrote ${remapAt} and ${downAt}`)
}

console.log(`
next:
  gameMap.maxIndex is now ${count} — update cmd/api/example.yaml and deploy/vps/backend.yaml
  cd ../backend && make map        # copy both blobs, then commit all three copies of each
  npm run borderLines              # traced from both blobs, stale until it is run
  npm run earth                    # the globe's texture, cut from the new tile field
  npm run landmassNames            # each landmass's name, keyed by the new borders blob
  npm run map:audit                # every fault should read zero`)
