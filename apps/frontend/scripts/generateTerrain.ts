import {fileURLToPath} from "node:url"

import sharp from "sharp"

import {latticeIds} from "../src/domain/lattice.ts"
import {LAND, OPEN_SEA, encodeTerrain} from "../src/domain/terrain.ts"
import {readBoard} from "./map/board.mjs"
import {landKinds, smoothed, terrainOf, tileColours} from "./map/terrain.mjs"
import {tileLattice} from "./map/tileLattice.mjs"
import {syncBoards, writeBoardFile, writeManifest} from "./writeBoard.ts"

// The satellite photo the game showed until Nov 2026, its coastline already cut to earth-1's tiles. Read here, never served.
const PHOTO = fileURLToPath(new URL("./map/earth.jpg", import.meta.url))

const board = readBoard()
const {data, info} = await sharp(PHOTO).raw().toBuffer({resolveWithObject: true})
const lattice = tileLattice(board)

const kinds = smoothed(landKinds(tileColours({data, ...info}, board.positions)), lattice, board.points)
const terrain = terrainOf(kinds, lattice, latticeIds(board.manifest.lattice.cols))
const bytes = encodeTerrain(terrain)

const file = writeBoardFile(board.manifest.name, "terrain", bytes)
const manifest = writeManifest(board.manifest.name)
syncBoards()

const shares = LAND.map((kind, k) => `${kind} ${(kinds.filter((each) => each === k).length / kinds.length * 100).toFixed(1)}%`)
const coast = terrain.filter((kind) => kind >= LAND.length && kind < OPEN_SEA).length
console.log(`wrote map/boards/${board.manifest.name}/${file} and ${manifest}: ${terrain.length} lattice points in`
    + ` ${(bytes.byteLength / 1024).toFixed(0)} KB
  ${board.count} tiles: ${shares.join(", ")}
  ${coast} points of coastal sea
next:
  cd ../backend && make map        # copy the board, then commit every copy
  cd ../backoffice && npm run map  # copy the manifest`)
