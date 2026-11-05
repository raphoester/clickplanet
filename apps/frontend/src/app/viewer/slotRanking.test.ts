import {describe, expect, it} from "vitest"
import {SLOT_COUNT, SlotRanking} from "./slotRanking.ts"

const needs = (entries: Record<string, number>) => new Map(Object.entries(entries))

describe("SlotRanking", () => {
    it("gives the flags holding the biggest landmasses a slot each, biggest first", () => {
        const ranking = new SlotRanking(3)

        expect(ranking.rank(needs({fr: 0.1, br: 0.6, de: 0.3, es: 0.2}))).toEqual(["br", "de", "es"])
    })

    it("leaves slots empty while fewer flags are painted", () => {
        const ranking = new SlotRanking(3)

        expect(ranking.rank(needs({fr: 0.1}))).toEqual(["fr", undefined, undefined])
        expect(ranking.rank(needs({}))).toEqual([undefined, undefined, undefined])
    })

    it("keeps a flag in its slot while it holds its place, whatever the others do", () => {
        const ranking = new SlotRanking(3)
        ranking.rank(needs({fr: 0.1, br: 0.6, de: 0.3}))

        expect(ranking.rank(needs({fr: 0.1, br: 0.6, de: 0.7}))).toEqual(["br", "de", "fr"])
        expect(ranking.rank(needs({fr: 0.5, br: 0.6, de: 0.7}))).toEqual(["br", "de", "fr"])
    })

    it("frees the slot of a flag painted nowhere any more, for the next in line", () => {
        const ranking = new SlotRanking(3)
        ranking.rank(needs({br: 0.6, de: 0.3, fr: 0.1, es: 0.05}))

        expect(ranking.rank(needs({br: 0.6, fr: 0.1, es: 0.05}))).toEqual(["br", "es", "fr"])
    })

    it("evicts the weakest only for a flag that needs clearly more", () => {
        const ranking = new SlotRanking(2, 1.2)
        ranking.rank(needs({br: 0.6, de: 0.3}))

        expect(ranking.rank(needs({br: 0.6, de: 0.3, es: 0.35}))).toEqual(["br", "de"])
        expect(ranking.rank(needs({br: 0.6, de: 0.3, es: 0.37}))).toEqual(["br", "es"])
    })

    it("holds eight by default", () => {
        expect(new SlotRanking().rank(needs({}))).toHaveLength(SLOT_COUNT)
    })
})
