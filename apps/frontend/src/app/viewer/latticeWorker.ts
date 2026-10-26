import {rebuildTilePositions} from "../../domain/landmask.ts"

export type LatticeAnswer = {positions: Float32Array} | {error: string}

self.onmessage = ({data}: MessageEvent<ArrayBuffer>) => {
    let answer: LatticeAnswer
    try {
        answer = {positions: rebuildTilePositions(data)}
    } catch (e) {
        answer = {error: e instanceof Error ? e.message : String(e)}
    }
    self.postMessage(answer, {transfer: "positions" in answer ? [answer.positions.buffer] : []})
}
