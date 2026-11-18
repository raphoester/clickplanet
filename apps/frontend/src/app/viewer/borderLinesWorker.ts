import {decodeBorderLines} from "../../domain/borderLinesFile.ts"
import {decodeProvinceLines} from "../../domain/provinceLinesFile.ts"
import {buffersOf, type OutlineSets, outlineSetsOf, type ProvinceOutline, provinceOutlineOf} from "../../domain/outlineSets.ts"

// file: the countries' lines or the provinces'; tolerances: the coarse sets' (coarseTolerancesOf).
export type OutlineQuestion = {kind: "countries" | "provinces", file: ArrayBuffer, tolerances: readonly number[]}

export type OutlineAnswer =
    | {kind: "countries", sets: OutlineSets}
    | {kind: "provinces", outline: ProvinceOutline}
    | {kind: "error", error: string}

self.onmessage = ({data}: MessageEvent<OutlineQuestion>) => {
    let answer: OutlineAnswer
    try {
        answer = data.kind === "countries"
            ? {kind: "countries", sets: outlineSetsOf(decodeBorderLines(data.file), data.tolerances)}
            : {kind: "provinces", outline: provinceOutlineOf(decodeProvinceLines(data.file), data.tolerances)}
    } catch (e) {
        answer = {kind: "error", error: e instanceof Error ? e.message : String(e)}
    }
    self.postMessage(answer, {transfer: transferOf(answer)})
}

function transferOf(answer: OutlineAnswer): ArrayBuffer[] {
    switch (answer.kind) {
        case "countries": return buffersOf(answer.sets)
        case "provinces": return [...new Set([...buffersOf(answer.outline.sets), answer.outline.countrySides.buffer as ArrayBuffer])]
        case "error": return []
    }
}
