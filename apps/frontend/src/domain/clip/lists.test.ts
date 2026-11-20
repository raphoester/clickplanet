import {describe, expect, it} from "vitest"
import {BoardScale} from "../boardScale.ts"
import {TileChange} from "./changes.ts"
import {Point} from "./geometry.ts"
import {aimedTilesOf, codeOf, entriesOf, ListGround, listsOf} from "./lists.ts"

// A big country is 10 tiles, and a painted flag reads from far at 1.
const SCALE: BoardScale = {length: 10, density: 0.01}

const at = (latitude: number, longitude: number): Point => {
    const phi = latitude * Math.PI / 180
    const lambda = longitude * Math.PI / 180
    return {x: Math.cos(phi) * Math.sin(lambda), y: Math.sin(phi), z: Math.cos(phi) * Math.cos(lambda)}
}

const COUNTRIES: [code: string, first: number, last: number, point: Point][] = [
    ["fr", 1, 20, at(46, 2)],
    ["tr", 21, 40, at(39, 35)],
    ["pw", 41, 42, at(7, 134)],
    ["id", 43, 62, at(-2, 118)],
    ["hu", 63, 82, at(47, 19)],
    ["at", 83, 102, at(47, 14)],
    ["aq", 103, 122, at(-80, 0)],
]

const countryOf = (tile: number) => COUNTRIES.find(([, first, last]) => tile >= first && tile <= last)

const held = (flag: string, first: number, last: number) =>
    Array.from({length: last - first + 1}, (_, i): [number, string] => [first + i, flag])

const ground: ListGround = {
    owners: new Map([
        ...held("pw", 1, 16), ...held("fr", 17, 18), ...held("fr", 21, 32), ...held("pw", 41, 42), ...held("hu", 43, 52),
        ...held("id", 53, 62), ...held("hu", 63, 82), ...held("hu", 83, 102),
    ]),
    tiles: 122,
    groundOf: (tile) => countryOf(tile)?.[0],
    pointOf: (tile) => countryOf(tile)![3],
}

const change = (tile: number, from: string | undefined, to: string | undefined): TileChange => ({tile, from, to, at: 0})

const attacks = [
    change(1, "fr", "pw"), change(2, "fr", "pw"), change(21, "tr", "fr"), change(19, "fr", undefined), change(20, undefined, "fr"),
    change(43, "id", "hu"), change(44, "id", "hu"), change(45, "id", "tr"),
]

describe("the carousel's lists", () => {
    const lists = listsOf(ground, attacks, SCALE)

    it("ranks the countries big enough to show by the share their own flag holds, the least first, then the most taken", () => {
        expect(lists.ownLand).toEqual([
            {country: "at", tiles: 20, ownShare: 0, holder: "hu", holderShare: 1},
            {country: "tr", tiles: 20, ownShare: 0, holder: "fr", holderShare: 0.6},
            {country: "fr", tiles: 20, ownShare: 0.1, holder: "pw", holderShare: 0.8},
            {country: "id", tiles: 20, ownShare: 0.5, holder: "hu", holderShare: 0.5},
            {country: "hu", tiles: 20, ownShare: 1, holder: "hu", holderShare: 1},
        ])
    })

    it("leaves out a country too small to show, and one nobody holds any of", () => {
        expect(lists.ownLand.map(({country}) => country)).not.toContain("pw")
        expect(lists.ownLand.map(({country}) => country)).not.toContain("aq")
    })

    it("ranks the flags by the tiles another flag took from them, with who took the most and where", () => {
        expect(lists.mostAttacked).toEqual([
            {flag: "id", lost: 3, takenBy: "hu", takenByTiles: 2, country: "id"},
            {flag: "fr", lost: 2, takenBy: "pw", takenByTiles: 2, country: "fr"},
            {flag: "tr", lost: 1, takenBy: "fr", takenByTiles: 1, country: "tr"},
        ])
    })

    it("ranks the flags by their land farthest from home, a big holding next door counting less than one far away", () => {
        expect(lists.farFromHome.map(({flag, country, tiles}) => ({flag, country, tiles}))).toEqual([
            {flag: "pw", country: "fr", tiles: 16},
            {flag: "hu", country: "id", tiles: 10},
            {flag: "fr", country: "tr", tiles: 12},
        ])
        expect(lists.farFromHome[1].km).toBeGreaterThan(9_000)
        expect(lists.farFromHome[1].km % 100).toBe(0)
    })

    it("ranks the flags by the tiles they hold of the world, with the country they hold the most of", () => {
        expect(lists.biggestEmpires).toEqual([
            {flag: "hu", tiles: 50, share: 50 / 122, country: "at"},
            {flag: "pw", tiles: 18, share: 18 / 122, country: "fr"},
            {flag: "fr", tiles: 14, share: 14 / 122, country: "tr"},
            {flag: "id", tiles: 10, share: 10 / 122, country: "id"},
        ])
    })

    it("names each entry by its country for own-land and its flag for the others", () => {
        expect(entriesOf(lists, "own-land").map(codeOf)).toEqual(["at", "tr", "fr", "id", "hu"])
        expect(entriesOf(lists, "most-attacked").map(codeOf)).toEqual(["id", "fr", "tr"])
    })

    it("aims each rank at its place: the country, the flag's land there, or what was taken from it there", () => {
        expect(aimedTilesOf(lists.ownLand[0], ground, attacks)).toEqual(held("x", 83, 102).map(([tile]) => tile))
        expect(aimedTilesOf(lists.farFromHome[0], ground, attacks)).toEqual(held("x", 1, 16).map(([tile]) => tile))
        expect(aimedTilesOf(lists.mostAttacked[1], ground, attacks)).toEqual([1, 2])
    })
})
