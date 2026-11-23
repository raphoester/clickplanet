import {describe, expect, it} from "vitest"
import {BoardScale} from "../boardScale.ts"
import {TileChange} from "./changes.ts"
import {Point} from "./geometry.ts"
import {aimedTilesOf, codeOf, entriesOf, ListGround, listsOf} from "./lists.ts"

// A painted flag reads from far at 1 tile.
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

const fights = [
    change(1, "fr", "pw"), change(2, "fr", "pw"), change(21, "tr", "fr"), change(19, "fr", undefined), change(20, undefined, "fr"),
    change(43, "id", "hu"), change(44, "id", "hu"), change(45, "id", "tr"),
]

describe("the carousel's lists", () => {
    const lists = listsOf(ground, fights, SCALE)

    it("ranks the flags by the tiles they took from other flags alone, a flag that only lost left out", () => {
        expect(lists.fightTheMost).toEqual([
            {flag: "hu", took: 2, lost: 0, rival: "id", rivalTiles: 2, country: "id"},
            {flag: "pw", took: 2, lost: 0, rival: "fr", rivalTiles: 2, country: "fr"},
            {flag: "fr", took: 1, lost: 2, rival: "pw", rivalTiles: 2, country: "fr"},
            {flag: "tr", took: 1, lost: 1, rival: "fr", rivalTiles: 1, country: "id"},
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

    it("ranks the flags by the distance their photos show, a holding under a painted flag read from far left out", () => {
        // A painted flag reads from far at 5 tiles.
        const coarse: BoardScale = {length: 10, density: 0.05}
        const far = listsOf({
            ...ground,
            owners: new Map([...held("fr", 48, 62), ...held("at", 103, 107), ...held("hu", 43, 46)]),
        }, [], coarse).farFromHome

        expect(far).toEqual([
            {flag: "at", country: "aq", tiles: 5, km: 14_100},
            {flag: "fr", country: "id", tiles: 15, km: 12_100},
        ])
    })

    it("ranks the flags by the tiles they hold of the world, with the country they hold the most of", () => {
        expect(lists.biggestEmpires).toEqual([
            {flag: "hu", tiles: 50, share: 50 / 122, country: "at"},
            {flag: "pw", tiles: 18, share: 18 / 122, country: "fr"},
            {flag: "fr", tiles: 14, share: 14 / 122, country: "tr"},
            {flag: "id", tiles: 10, share: 10 / 122, country: "id"},
        ])
    })

    it("names each entry by its flag", () => {
        expect(entriesOf(lists, "fight-the-most").map(codeOf)).toEqual(["hu", "pw", "fr", "tr"])
        expect(entriesOf(lists, "far-from-home").map(codeOf)).toEqual(["pw", "hu", "fr"])
    })

    it("aims each rank at its place: the flag's land there, or the tiles it fought for there", () => {
        expect(aimedTilesOf(lists.farFromHome[0], ground, fights)).toEqual(held("x", 1, 16).map(([tile]) => tile))
        expect(aimedTilesOf(lists.fightTheMost[2], ground, fights)).toEqual([1, 2])
    })
})
