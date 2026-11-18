import {type Lattice, rebuildTiles, type RebuiltTiles} from "../../domain/board.ts"

// `hash` is the one the board's manifest says.
export type LatticeQuestion = {lattice: Lattice, tiles: number, runs: ArrayBuffer, hash: number}

export type LatticeAnswer = RebuiltTiles | {error: string}

self.onmessage = ({data}: MessageEvent<LatticeQuestion>) => {
    let answer: LatticeAnswer
    try {
        answer = rebuildTiles(data.lattice, data.tiles, data.runs, data.hash)
    } catch (e) {
        answer = {error: e instanceof Error ? e.message : String(e)}
    }
    const transfer = "positions" in answer
        ? [answer.positions, answer.cells.words, answer.cells.texels, answer.cells.tiles, answer.chunks.starts, answer.chunks.caps, answer.chunks.blocks].map(({buffer}) => buffer)
        : []
    self.postMessage(answer, {transfer})
}
