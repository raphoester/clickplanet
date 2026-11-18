import {decodeBorderLines} from "../../domain/borderLinesFile.ts"
import {buffersOf, type OutlineSets, outlineSetsOf} from "../../domain/outlineSets.ts"

// tolerances: the coarse sets' (coarseTolerancesOf).
export type OutlineQuestion = {file: ArrayBuffer, tolerances: readonly number[]}

export type OutlineAnswer = OutlineSets | {error: string}

self.onmessage = ({data}: MessageEvent<OutlineQuestion>) => {
    let answer: OutlineAnswer
    try {
        answer = outlineSetsOf(decodeBorderLines(data.file), data.tolerances)
    } catch (e) {
        answer = {error: e instanceof Error ? e.message : String(e)}
    }
    self.postMessage(answer, {transfer: "cells" in answer ? buffersOf(answer) : []})
}
