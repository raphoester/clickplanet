import {rebuildTilePositions} from "../../domain/landmask.ts"
import {cellsOf, type TileCells} from "../../domain/latticeCells.ts"

export type LatticeAnswer = {positions: Float32Array, cells: TileCells} | {error: string}

self.onmessage = ({data}: MessageEvent<ArrayBuffer>) => {
    let answer: LatticeAnswer
    try {
        const positions = rebuildTilePositions(data)
        answer = {positions, cells: cellsOf(positions)}
    } catch (e) {
        answer = {error: e instanceof Error ? e.message : String(e)}
    }
    const transfer = "positions" in answer ? [answer.positions.buffer, answer.cells.words.buffer, answer.cells.texels.buffer] : []
    self.postMessage(answer, {transfer})
}
