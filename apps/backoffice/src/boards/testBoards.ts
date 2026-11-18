import {readdirSync, readFileSync} from "node:fs"
import {join} from "node:path"
import {Board, boardOf} from "./boards.ts"
import {boardPlaces, Places} from "./places.ts"

// For the tests: earth-1, bundled, and fixture-1, the tests' second board, a coarse earth at cols 31 (2,998 tiles,
// about 9.7 times as far apart). fixture-1 is read from src/testdata alone: no page bundles it.
// import.meta.dirname, not new URL(…, import.meta.url): a jsdom test's transform makes that a served address.
const GEN = join(import.meta.dirname, "..", "gen", "map") + "/"
const TESTDATA = join(import.meta.dirname, "..", "testdata", "boards") + "/"

export const EARTH_1: Board = boardOf("earth-1").board
export const FIXTURE_1: Board = manifestIn(`${TESTDATA}fixture-1/`)

function manifestIn(directory: string): Board {
    const [file] = readdirSync(directory).filter(name => /^board-[0-9a-f]{8}\.json$/.test(name))
    return JSON.parse(readFileSync(directory + file, "utf8")) as Board
}

export function runsOf(board: Board): ArrayBuffer {
    const directory = board === FIXTURE_1 ? `${TESTDATA}fixture-1/` : `${GEN}${board.name}/`
    const bytes = readFileSync(directory + board.files.runs)
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)
}

const rebuilt = new Map<Board, Places>()

// A board's places, rebuilt from its runs once a test file.
export function placesOn(board: Board): Places {
    let places = rebuilt.get(board)
    if (places === undefined) {
        places = boardPlaces(board, runsOf(board))
        rebuilt.set(board, places)
    }
    return places
}
