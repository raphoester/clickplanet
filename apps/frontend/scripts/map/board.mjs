import fs from "node:fs"
import path from "node:path"
import {fileURLToPath} from "node:url"

import {decodeRuns, tilesOf} from "../../src/domain/board.ts"
import {latticeSize, positionsHash} from "../../src/domain/lattice.ts"
import {decodeProvinces} from "../../src/domain/provinces.ts"
import {decodeTerrain} from "../../src/domain/terrain.ts"
import {rotatedCorners} from "./warp.mjs"

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..")

export const mapDir = path.resolve(frontendRoot, "..", "..", "map")
export const boardsDir = path.join(mapDir, "boards")
export const staticDir = path.join(frontendRoot, "static")

// Today's lattice, with no warp and no rotation, its tiles numbered in three's order: the ledger holds those ids.
export const EARTH_1 = {name: "earth-1", body: "earth", lattice: {cols: 301, warp: null, corners: null, order: "three"}}

// The world's next board, about 1,000,000 tiles 12.5 km apart: the warp fitted at its `cols` (`fitWarp`, pinned in
// warp.test.mjs), the corners turned so the twelve five-neighbour points are at sea, its tiles in canonical order. A
// province is 200 tiles or more, unless it is a whole landmass or a whole UK nation (docs/provinces.md).
export const EARTH_2 = {
    name: "earth-2",
    body: "earth",
    lattice: {cols: 590, warp: {a: -0.09142428, b: -0.13039461, c: 0.26096331}, corners: rotatedCorners(), order: "canonical"},
    provinceFloor: 200,
}

// Capital Siege's board, about 27,000 tiles 79 km apart (the owner, 2026-11-17): 15 players at a click a second fill it
// in about 30 minutes with no spread. Its provinces group more than earth-2's: 20 tiles are about Bulgaria.
export const SIEGE_1 = {
    name: "siege-1",
    body: "earth",
    lattice: {cols: 97, warp: {a: -0.08982266, b: -0.13216128, c: 0.26291902}, corners: rotatedCorners(), order: "canonical"},
    provinceFloor: 20,
}

// Every board the generator writes, by name: `npm run map:generate -- earth-2`.
export const BOARD_DEFINITIONS = Object.fromEntries([EARTH_1, EARTH_2, SIEGE_1].map((board) => [board.name, board]))

/**
 * The board a script is asked for, by name among its arguments: earth-1 when none is.
 * @param {readonly string[]} args
 */
export function boardAsked(args) {
    const names = args.filter((arg) => !arg.startsWith("-") && arg in BOARD_DEFINITIONS)
    if (names.length > 1) throw new Error(`one board at a time: ${names.join(", ")}`)
    return BOARD_DEFINITIONS[names[0] ?? EARTH_1.name]
}

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

/** @returns {{name: string, provinces: import("../../src/domain/provinces.ts").Provinces}} */
export function readProvinces(board = readBoard()) {
    const name = board.manifest.files.provinces
    if (name === undefined) throw new Error(`${board.file} names no provinces: run \`npm run provinces -- ${board.manifest.name}\``)
    const bytes = fs.readFileSync(path.join(board.directory, name))
    return {name, provinces: decodeProvinces(bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength), board.count)}
}

/** @returns {{name: string, terrain: Uint8Array}} each lattice point's kind, by canonical id */
export function readTerrain(board = readBoard()) {
    const name = board.manifest.files.terrain
    const bytes = fs.readFileSync(path.join(board.directory, name))
    const buffer = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)
    return {name, terrain: decodeTerrain(buffer, latticeSize(board.manifest.lattice.cols))}
}
