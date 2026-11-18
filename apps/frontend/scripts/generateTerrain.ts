import {fileURLToPath} from "node:url"

import sharp from "sharp"

import {latticeIds} from "../src/domain/lattice.ts"
import {cellsOf, latticeFaces, tileAt} from "../src/domain/latticeCells.ts"
import {LAND, OPEN_SEA, encodeTerrain} from "../src/domain/terrain.ts"
import {EARTH_1, boardAsked, boardDir, readBoard, readTerrain} from "./map/board.mjs"
import {kindsFromNeighbours, landKinds, smoothed, terrainOf, tileColours} from "./map/terrain.mjs"
import {tileLattice} from "./map/tileLattice.mjs"
import {syncBoards, writeBoardFile, writeManifest} from "./writeBoard.ts"

// The satellite photo the game showed until Nov 2026, its coastline already cut to earth-1's tiles. Read here, never served.
const PHOTO = fileURLToPath(new URL("./map/earth.jpg", import.meta.url))

// `npm run terrain -- earth-2`; earth-1 when no board is named.
const board = readBoard(boardDir(boardAsked(process.argv.slice(2)).name))
const {data, info} = await sharp(PHOTO).raw().toBuffer({resolveWithObject: true})
const lattice = tileLattice(board)

// The photo's sea is earth-1's: a tile whose point is in a cell of earth-1 that is no tile reads blue, and takes its
// kind from its tiles around; an island earth-1 did not have, the kind of earth-1's nearest tile. None on earth-1.
const earth1 = readBoard(boardDir(EARTH_1.name))
const earth1Cells = cellsOf(earth1.ids, earth1.manifest.lattice, earth1.positions)
const earth1Faces = latticeFaces(earth1.manifest.lattice)
const astray = new Uint8Array(board.count)
for (let t = 0; t < board.count; t++) {
    const at = t * 3
    const under = tileAt(earth1Cells, earth1Faces, board.positions[at], board.positions[at + 1], board.positions[at + 2])
    if (under === undefined) astray[t] = 1
}
const earth1Terrain = readTerrain(earth1).terrain
const nearestEarth1Kind = (tile: number): number => {
    let best = -2, kind = 0
    for (let e = 0; e < earth1.count; e++) {
        const dot = board.positions[tile * 3] * earth1.positions[e * 3] + board.positions[tile * 3 + 1] * earth1.positions[e * 3 + 1]
            + board.positions[tile * 3 + 2] * earth1.positions[e * 3 + 2]
        if (dot > best) {
            best = dot
            kind = earth1Terrain[earth1.ids[e]]
        }
    }
    return kind
}

const photoKinds = landKinds(tileColours({data, ...info}, board.positions))
const filled = kindsFromNeighbours(photoKinds, astray, lattice, board.points, nearestEarth1Kind)
const kinds = smoothed(photoKinds, lattice, board.points)
const terrain = terrainOf(kinds, lattice, latticeIds(board.manifest.lattice))
const bytes = encodeTerrain(terrain)

const file = writeBoardFile(board.manifest.name, "terrain", bytes)
const manifest = writeManifest(board.manifest.name)
syncBoards()

const shares = LAND.map((kind, k) => `${kind} ${(kinds.filter((each) => each === k).length / kinds.length * 100).toFixed(1)}%`)
const coast = terrain.filter((kind) => kind >= LAND.length && kind < OPEN_SEA).length
console.log(`wrote map/boards/${board.manifest.name}/${file} and ${manifest}: ${terrain.length} lattice points in`
    + ` ${(bytes.byteLength / 1024).toFixed(0)} KB
  ${board.count} tiles: ${shares.join(", ")}
  ${filled.neighbours} tiles where earth-1 had sea took their kind from their land neighbours, ${filled.fallback} from earth-1's nearest tile
  ${coast} points of coastal sea
next:
  cd ../backend && make map        # copy the board, then commit every copy
  cd ../backoffice && npm run map  # copy the manifest`)
