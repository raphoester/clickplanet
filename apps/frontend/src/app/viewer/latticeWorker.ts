import {type Lattice, rebuildTiles, type RebuiltTiles} from "../../domain/board.ts"

export type LatticeQuestion = {lattice: Lattice, tiles: number, runs: ArrayBuffer}

export type LatticeAnswer = RebuiltTiles | {error: string}

self.onmessage = ({data}: MessageEvent<LatticeQuestion>) => {
    let answer: LatticeAnswer
    try {
        answer = rebuildTiles(data.lattice, data.tiles, data.runs)
    } catch (e) {
        answer = {error: e instanceof Error ? e.message : String(e)}
    }
    const transfer = "positions" in answer ? [answer.positions.buffer, answer.cells.words.buffer, answer.cells.texels.buffer] : []
    self.postMessage(answer, {transfer})
}
