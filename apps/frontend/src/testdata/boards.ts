import {readdirSync, readFileSync} from "node:fs"
import {dirname, resolve} from "node:path"
import {fileURLToPath} from "node:url"
import {type Board, type BoardAsset, rebuildTiles, type RebuiltTiles} from "../domain/board.ts"
import {boardOf} from "../app/viewer/boards.ts"

// From the module's own path, as a string: jsdom's URL is not one node takes.
const HERE = dirname(fileURLToPath(import.meta.url))
const STATIC = `${resolve(HERE, "..", "..", "static")}/`
const TESTDATA = `${resolve(HERE, "boards")}/`

// The boards the tests read: earth-1 as the page serves it, and fixture-1, a coarse earth cut from it that no page
// serves (`npm run map:fixture`): about ten times as far apart, a hundredth as dense.
export const EARTH_1: BoardAsset = boardOf("earth-1")

export const FIXTURE_1: Board = manifestIn(`${TESTDATA}fixture-1/`)

function manifestIn(directory: string): Board {
    const [file] = readdirSync(directory).filter((name) => /^board-[0-9a-f]{8}\.json$/.test(name))
    if (file === undefined) throw new Error(`no manifest in ${directory}`)
    return JSON.parse(readFileSync(directory + file, "utf8")) as Board
}

function directoryOf(board: Board): string {
    return board.name === FIXTURE_1.name ? `${TESTDATA}${board.name}/` : `${STATIC}boards/${board.name}/`
}

// One of a board's files, as the page would fetch it.
export function bytesOf(board: Board, file: string): ArrayBuffer {
    const bytes = readFileSync(directoryOf(board) + file)
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)
}

const rebuilt = new Map<string, RebuiltTiles>()

// A board's tiles, rebuilt from its runs as the page rebuilds them.
export function tilesOfBoard(board: Board): RebuiltTiles {
    let tiles = rebuilt.get(board.name)
    if (tiles === undefined) {
        tiles = rebuildTiles(board.lattice, board.tiles, bytesOf(board, board.files.runs), board.positionsHash)
        rebuilt.set(board.name, tiles)
    }
    return tiles
}

// A board's landmass names in English, as the page fetches them beside the board.
export function englishLandmassNamesOf(board: BoardAsset): Record<string, string> {
    return (JSON.parse(readFileSync(`${directoryOf(board)}${board.page.landmassNames}`, "utf8")) as {names: Record<string, string>}).names
}
