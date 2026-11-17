import fs from "node:fs"
import path from "node:path"
import {fileURLToPath} from "node:url"

import {decodeRuns, tilesOf} from "../../src/domain/board.ts"
import {positionsHash} from "../../src/domain/lattice.ts"

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..")

export const mapDir = path.resolve(frontendRoot, "..", "..", "map")
export const boardsDir = path.join(mapDir, "boards")
export const staticDir = path.join(frontendRoot, "static")

// Today's lattice, with no warp and no rotation, its tiles numbered in three's order: the ledger holds those ids.
export const EARTH_1 = {name: "earth-1", body: "earth", lattice: {cols: 301, warp: null, corners: null, order: "three"}}

const MANIFEST = /^board-[0-9a-f]{8}\.json$/

export function blobNamed(pattern, directory) {
    const names = fs.readdirSync(directory).filter((entry) => pattern.test(entry))
    if (names.length !== 1) {
        throw new Error(`expected one ${directory}/${pattern.source}, found ${names.length}`)
    }
    return names[0]
}

export const boardDir = (name = EARTH_1.name) => path.join(boardsDir, name)

/** @returns {{file: string, manifest: import("../../src/domain/board.ts").Board}} */
export function readManifest(directory = boardDir()) {
    const file = blobNamed(MANIFEST, directory)
    return {file, manifest: JSON.parse(fs.readFileSync(path.join(directory, file), "utf8"))}
}

/**
 * A board, its tiles rebuilt from its runs and checked against its manifest's hash.
 * @returns {{
 *   directory: string,
 *   file: string,
 *   manifest: import("../../src/domain/board.ts").Board,
 *   count: number,
 *   positions: Float32Array,
 *   ids: Uint32Array,
 *   points: Uint32Array,
 * }}
 */
export function readBoard(directory = boardDir()) {
    const {file, manifest} = readManifest(directory)
    const bytes = fs.readFileSync(path.join(directory, manifest.files.runs))
    const runs = decodeRuns(bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength))
    const tiles = tilesOf(manifest.lattice, runs, manifest.tiles)
    const hash = positionsHash(tiles.positions)
    if (hash !== manifest.positionsHash) {
        throw new Error(`${file}: its runs rebuild positions that hash to ${hash}, it says ${manifest.positionsHash}`)
    }
    return {directory, file, manifest, count: manifest.tiles, ...tiles}
}

/**
 * Each lattice point's tile, or -1 at sea.
 * @param {{count: number, points: Uint32Array}} board
 * @param {number} vertices the lattice's point count
 */
export function tileOfPoint({count, points}, vertices) {
    const tileOf = new Int32Array(vertices).fill(-1)
    for (let t = 0; t < count; t++) tileOf[points[t]] = t
    return tileOf
}

/** @returns {{name: string, count: number, codes: string[], landmass: Uint16Array}} */
export function readBorders(board = readBoard()) {
    const name = board.manifest.files.borders
    const bytes = fs.readFileSync(path.join(board.directory, name))
    const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)

    const headerLength = view.getUint32(0, true)
    const {tiles, codes} = JSON.parse(bytes.subarray(4, 4 + headerLength).toString("utf8"))

    return {
        name,
        count: tiles,
        codes,
        landmass: new Uint16Array(bytes.buffer, bytes.byteOffset + 4 + headerLength, tiles),
    }
}

/** @returns {{name: string, tiles: number, capitals: Record<string, number>}} */
export function readCapitals(board = readBoard()) {
    const name = board.manifest.files.capitals
    return {name, ...JSON.parse(fs.readFileSync(path.join(board.directory, name), "utf8"))}
}

/** @returns {{name: string, tiles: number, bridges: [number, number][]}} */
export function readBridges(board = readBoard()) {
    const name = board.manifest.files.bridges
    return {name, ...JSON.parse(fs.readFileSync(path.join(board.directory, name), "utf8"))}
}
