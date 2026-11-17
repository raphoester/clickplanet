import {describe, expect, it} from "vitest"
import {joinSpans, spansOf} from "./spans.ts"

describe("spansOf", () => {
    it("makes one run of consecutive values, and another past a gap", () => {
        expect(spansOf([2, 3, 4, 9])).toEqual([{start: 2, count: 3}, {start: 9, count: 1}])
    })

    it("counts a value seen twice once", () => {
        expect(spansOf([5, 5, 6])).toEqual([{start: 5, count: 2}])
    })

    it("joins values whose width reaches the next one", () => {
        expect(spansOf([0, 4, 8, 16], 4)).toEqual([{start: 0, count: 12}, {start: 16, count: 4}])
    })

    it("makes nothing of nothing", () => {
        expect(spansOf([])).toEqual([])
    })
})

describe("joinSpans", () => {
    const spans = [{start: 0, count: 2}, {start: 3, count: 1}, {start: 10, count: 2}, {start: 14, count: 1}]

    it("keeps the spans when there are few enough", () => {
        expect(joinSpans(spans, 4)).toEqual(spans)
    })

    it("joins across the shortest gaps first", () => {
        expect(joinSpans(spans, 2)).toEqual([{start: 0, count: 4}, {start: 10, count: 5}])
        expect(joinSpans(spans, 1)).toEqual([{start: 0, count: 15}])
    })

    it("joins every gap under the least it is given, however few the spans", () => {
        expect(joinSpans(spans, 4, 3)).toEqual([{start: 0, count: 4}, {start: 10, count: 5}])
    })

    it("keeps the earlier of two gaps as wide", () => {
        const even = [{start: 0, count: 1}, {start: 2, count: 1}, {start: 4, count: 1}]
        expect(joinSpans(even, 2)).toEqual([{start: 0, count: 1}, {start: 2, count: 3}])
    })

    it("hands back spans of its own", () => {
        const joined = joinSpans(spans, 4)
        joined[0].count = 99
        expect(spans[0].count).toBe(2)
    })

    it("makes nothing of nothing", () => {
        expect(joinSpans([], 3)).toEqual([])
    })
})
