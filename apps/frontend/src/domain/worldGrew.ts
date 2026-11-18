// The world's switch of board (docs/boards.md): every browser that saw the world before it sees "The world grew" once,
// on the first page it draws on the new board. Never on a first visit, which never saw the world smaller, and never on
// another board: an arena's plays a board of its own.

export const WORLD_GREW_STORAGE_KEY = "clickplanet-world-grew"

// The board the world played before the switch, and the one it plays after.
export const WORLD_BEFORE = "earth-1"
export const WORLD_GROWN = "earth-2"

export type WorldGrew = {
    board: string
    // How many times the ground: 4 from earth-1's 260,503 tiles to earth-2's 1,004,682.
    times: number
}

type Sized = {name: string, tiles: number}

// What a page drawing board shows, with the board name the browser kept (seen) and whether this is its first visit.
// A first visit is told nothing, and keeps the board as seen.
export function worldGrewOn(board: Sized, before: Sized, seen: string | null, firstVisit: boolean): WorldGrew | undefined {
    if (board.name !== WORLD_GROWN || seen === WORLD_GROWN || firstVisit) return undefined
    return {board: board.name, times: Math.round(board.tiles / before.tiles)}
}

// Whether the browser keeps the board as seen now: once the moment is shown, or on a first visit to it.
export function worldGrewKept(board: Sized, seen: string | null, firstVisit: boolean): boolean {
    return board.name === WORLD_GROWN && seen !== WORLD_GROWN && firstVisit
}
