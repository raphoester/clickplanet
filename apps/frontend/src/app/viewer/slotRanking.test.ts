import {describe, expect, it} from "vitest"
import {type FlagNeed, joinedNeeds, rankSlots, SlotLayout} from "./slotRanking.ts"

const BUDGET = {roots: 1, largest: 1}

const flag = (need: number, area = 1e6, detail = 1): FlagNeed => ({need, area, aspect: 1.5, spriteWidth: 100, detail})

const needs = (entries: Record<string, FlagNeed>) => new Map(Object.entries(entries))

const levels = (ranked: {code: string, level: number}[]) => Object.fromEntries(ranked.map(({code, level}) => [code, level]))

describe("joinedNeeds", () => {
    it("asks for a flag in both as wide as its widest need, with all its land", () => {
        const joined = joinedNeeds(needs({fr: flag(300, 10), de: flag(200, 5)}), needs({fr: flag(500, 20), it: flag(150, 1)}))
        expect(Object.fromEntries(joined)).toEqual({fr: flag(500, 30), de: flag(200, 5), it: flag(150, 1)})
    })
})

describe("rankSlots", () => {
    it("leaves a flag no wider on screen than its sprite on the atlas", () => {
        expect(rankSlots(needs({fr: flag(120)}), new Map(), BUDGET)).toEqual([])
    })

    it("gives a flag in view the smallest cell that holds it as wide as it needs", () => {
        expect(rankSlots(needs({fr: flag(300)}), new Map(), BUDGET)).toEqual([{code: "fr", level: 4, least: 6}])
    })

    it("holds a flag with little land on screen to a cell its land pays for, however wide its frame", () => {
        expect(levels(rankSlots(needs({cl: flag(1500, 10_000), br: flag(1500)}), new Map(), BUDGET))).toEqual({br: 1, cl: 4})
    })

    it("shares a full texture by land, and raises the most land it has room for to the share above", () => {
        const ranked = rankSlots(needs({ru: flag(2000, 600_000), kz: flag(2000, 300_000), ua: flag(2000, 100_000)}), new Map(), BUDGET)

        expect(ranked.map(({code}) => code)).toEqual(["ru", "kz", "ua"])
        expect(levels(ranked)).toEqual({ru: 1, kz: 2, ua: 3})
    })

    it("pays a flag of fine detail more texels for its land than a flag of bands", () => {
        const ranked = rankSlots(needs({ru: flag(2000, 600_000), sa: flag(400, 30_000, 2), pl: flag(400, 30_000, 0.35)}), new Map(), BUDGET)

        expect(levels(ranked)).toEqual({ru: 1, sa: 3, pl: 4})
    })

    it("keeps a flag's cell while it needs neither clearly more nor far less", () => {
        const held = new Map([["fr", 4]])

        expect(levels(rankSlots(needs({fr: flag(380)}), held, BUDGET))).toEqual({fr: 4})
        expect(levels(rankSlots(needs({fr: flag(500)}), held, BUDGET))).toEqual({fr: 2})
        expect(levels(rankSlots(needs({fr: flag(400)}), new Map([["fr", 2]]), BUDGET))).toEqual({fr: 2})
        expect(levels(rankSlots(needs({fr: flag(300)}), new Map([["fr", 2]]), BUDGET))).toEqual({fr: 4})
    })

    it("keeps a flag out of view in its cell while there is room, and drops it first", () => {
        const held = new Map([["br", 1]])

        expect(levels(rankSlots(needs({br: flag(0, 0), fr: flag(700)}), held, BUDGET))).toEqual({br: 1, fr: 2})
        expect(levels(rankSlots(needs({fr: flag(700)}), held, BUDGET))).toEqual({br: 1, fr: 2})
        expect(levels(rankSlots(needs({br: flag(0, 0), fr: flag(2000), de: flag(2000)}), held, BUDGET))).toEqual({fr: 1, de: 1})
    })
})

describe("SlotLayout", () => {
    it("keeps a flag's cell while its size holds", () => {
        const layout = new SlotLayout(BUDGET)
        const first = layout.arrange(needs({fr: flag(300), de: flag(450)}))

        expect(layout.arrange(needs({fr: flag(320), de: flag(440)}))).toEqual(first)
    })

    it("frees the cell of a flag gone for one that comes", () => {
        const layout = new SlotLayout(BUDGET)
        layout.arrange(needs({ru: flag(2000), kz: flag(2000)}))

        expect(layout.arrange(needs({ru: flag(2000), br: flag(2000), kz: flag(0, 0)})))
            .toEqual([
                {code: "ru", cell: {x: 1024, y: 0, width: 1024, height: 1024}},
                {code: "br", cell: {x: 0, y: 0, width: 1024, height: 1024}},
            ])
    })

    it("drops a size for a flag the free cells are too broken up for", () => {
        const layout = new SlotLayout(BUDGET)
        layout.arrange(needs({a: flag(450, 400_000), b: flag(450, 300_000), c: flag(450, 200_000), d: flag(450, 100_000)}))

        const placed = layout.arrange(needs({a: flag(450, 400_000), d: flag(450, 100_000), e: flag(2000, 2e6), f: flag(700, 1.5e6)}))

        expect(placed).toEqual([
            {code: "e", cell: {x: 1024, y: 0, width: 1024, height: 1024}},
            {code: "f", cell: {x: 512, y: 0, width: 512, height: 512}},
            {code: "a", cell: {x: 0, y: 0, width: 512, height: 512}},
            {code: "d", cell: {x: 512, y: 512, width: 512, height: 512}},
        ])
    })
})
