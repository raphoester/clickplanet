import {BOARDS as BUNDLED} from "../gen/map/boards.ts"
import {Lattice} from "./lattice.ts"

// A board's manifest (/map/README.md#format): the lattice it is cut from, how many tiles it has, how far apart, and
// the hash of their positions.
export type Board = {
    name: string
    body: string
    lattice: Lattice
    tiles: number
    // The median arc between two touching tiles, in radians.
    spacing: number
    positionsHash: number
    files: {runs: string} & Record<string, string>
}

// A board the page bundles: its manifest, the address its runs are served at, and its capitals file as text.
export type Bundled = {board: Board, runsUrl: string, capitals: string}

// Every board in /map/boards, by name, as `npm run map` copied them into src/gen/map.
export const BOARDS: ReadonlyMap<string, Bundled> = new Map(
    Object.entries(BUNDLED).map(([name, {manifest, runs, capitals}]): [string, Bundled] => [
        name, {board: manifest, runsUrl: runs, capitals},
    ]),
)

// The board an answer names. One the page does not bundle is refused: its tile ids are places on no map the page has.
export function boardOf(name: string): Bundled {
    const bundled = BOARDS.get(name)
    if (bundled !== undefined) return bundled
    if (name === "") throw new Error("the answer names no board")
    throw new Error(`${name} is not a board this page carries (it has ${[...BOARDS.keys()].join(", ")})`)
}
