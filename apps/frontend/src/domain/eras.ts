import {FIRST_BOARD} from "./board.ts"
import {MapHistory, MapReel} from "./mapHistory.ts"

// An era: a stretch of a map on one board, from startedAt on (planet's BoardEra). A switch of board starts one.
export type Era = {startedAt: number, board: string}

// What an answer that names no era is of: the first board, since the epoch.
export const FIRST_ERA: Era = {startedAt: 0, board: FIRST_BOARD}

// The era in force at a time, by its place in eras: the last to start at it or before, the first for a time before
// them all.
export function eraAt(eras: readonly Era[], at: number): number {
    let era = 0
    while (era + 1 < eras.length && eras[era + 1].startedAt <= at) era++
    return era
}

// A history's stretch on one board: its frames from `from` to `to` (excluded) and the hits of its era, as a history of
// its own, whose first frame is a whole map of its board: the window's first frame, or the cut. startedAt: when the
// stretch starts, the window's start for the first, the era's start for the next.
export type EraPiece = {era: number, board: string, from: number, to: number, startedAt: number, history: MapHistory}

// Each era's frames, in order: a frame is of the era at its time. An era with no frame in the window has no piece.
export function piecesOf(history: MapHistory): EraPiece[] {
    const {frames, eras} = history
    const pieces: EraPiece[] = []
    frames.forEach((frame, i) => {
        const era = eraAt(eras, frame.at)
        const last = pieces.at(-1)
        if (last !== undefined && last.era === era) {
            last.to = i + 1
            return
        }
        pieces.push({
            era,
            board: eras[era]?.board ?? FIRST_BOARD,
            from: i,
            to: i + 1,
            startedAt: last === undefined ? frame.at : Math.min(frame.at, eras[era]?.startedAt ?? frame.at),
            history: history,
        })
    })
    return pieces.map((piece) => ({
        ...piece,
        history: {
            frames: frames.slice(piece.from, piece.to),
            hits: [history.hits[piece.era] ?? new Map()],
            eras: [eras[piece.era] ?? FIRST_ERA],
        },
    }))
}

// The history from the start of its last era on, a history of one board: the map the window ends on.
export function lastEraOf(history: MapHistory): MapHistory & {board: string} {
    const last = piecesOf(history).at(-1)
    return last === undefined
        ? {...history, board: history.eras.at(-1)?.board ?? FIRST_BOARD}
        : {...last.history, board: last.board}
}

// Where the map is cleared and drawn whole on the next board: the first frame of each era after the first.
export type Cut = {frame: number, at: number, board: string}

// Any frame's map over several boards: each era's frames on its own reel, of its own board's tiles.
export class ErasReel {
    readonly pieces: readonly (EraPiece & {reel: MapReel})[]
    private readonly frames: MapHistory["frames"]

    // tilesOf: how many tiles a board has.
    constructor(history: MapHistory, tilesOf: (board: string) => number) {
        this.frames = history.frames
        this.pieces = piecesOf(history).map((piece) => ({...piece, reel: new MapReel(piece.history, tilesOf(piece.board))}))
    }

    get length(): number {
        return this.frames.length
    }

    atOf(frame: number): number {
        return this.frames[frame]?.at ?? 0
    }

    // The piece a frame is drawn on, and the frame's place in it.
    pieceAt(frame: number): {piece: ErasReel["pieces"][number], frame: number} {
        const at = Math.max(0, Math.min(this.length - 1, frame))
        const piece = this.pieces.find(({from, to}) => at >= from && at < to) ?? this.pieces[0]
        return {piece, frame: at - piece.from}
    }

    cuts(): Cut[] {
        return this.pieces.slice(1).map(({from, startedAt, board}) => ({frame: from, at: startedAt, board}))
    }
}
