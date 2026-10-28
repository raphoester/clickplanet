import {describe, expect, it} from "vitest"
import {TileChange} from "./changes.ts"
import {frontFactsOf, Ground, holdingStoryOf, placesOf, tourFactsOf} from "./findings.ts"
import {Point} from "./geometry.ts"

// Tiles 1 to 6 are France, 7 to 10 Germany, 11 and 12 Algeria.
const GROUND = new Map<number, string>([
    ...[1, 2, 3, 4, 5, 6].map((tile): [number, string] => [tile, "fr"]),
    ...[7, 8, 9, 10].map((tile): [number, string] => [tile, "de"]),
    ...[11, 12].map((tile): [number, string] => [tile, "dz"]),
])

const REGIONS: Record<string, string> = {fr: "Europe", de: "Europe", dz: "Africa", pw: "Oceania"}

const ground: Ground = {
    groundOf: (tile) => GROUND.get(tile),
    tilesOf: (country) => [...GROUND].flatMap(([tile, ground]) => ground === country ? [tile] : []),
    regionOf: (country) => REGIONS[country],
}

// Every tile at its own point, far from the others: solidity is not what these tests measure.
const pointOf = (tile: number): Point => ({x: Math.cos(tile), y: Math.sin(tile), z: 0})

const change = (tile: number, from: string | undefined, to: string | undefined, at = 0): TileChange => ({tile, from, to, at})

describe("the facts of a front", () => {
    const opening = new Map<number, string>([[1, "fr"], [2, "fr"], [3, "fr"], [4, "fr"], [7, "de"], [8, "de"], [11, "dz"]])
    const changes = [
        change(1, "fr", "pw"), change(2, "fr", "pw"), change(3, "fr", "pw"),
        change(7, "de", "pw"), change(9, "de", "pw"), change(8, "de", "fr"),
        change(5, undefined, "fr"), change(11, "dz", "dz"),
    ]
    const after = new Map<number, string>([[1, "pw"], [2, "pw"], [3, "pw"], [4, "fr"], [5, "fr"], [7, "pw"], [8, "fr"], [9, "pw"], [11, "dz"]])
    const flips = [
        {landmass: 1, tiles: 300, holders: ["fr", "pw"], tile: 1},
        {landmass: 2, tiles: 150, holders: ["de", "fr"], tile: 8},
        {landmass: 3, tiles: 60, holders: ["de", "pw"], tile: 9},
    ]

    const facts = frontFactsOf(changes, opening, after, ground, pointOf, flips)

    it("counts what changed hands and what was taken from another flag", () => {
        expect(facts.tilesChanged).toBe(7)
        expect(facts.captured).toBe(6)
    })

    it("lists each country with who held it when the front opened and when it ended, and who took there", () => {
        expect(facts.countries.map(({code, continent, tiles, changed}) => ({code, continent, tiles, changed}))).toEqual([
            {code: "fr", continent: "Europe", tiles: 6, changed: 4},
            {code: "de", continent: "Europe", tiles: 4, changed: 3},
        ])
        expect(facts.countries[0].holdersStart).toEqual({fr: 4})
        expect(facts.countries[0].holdersEnd).toEqual({pw: 3, fr: 2})
        expect(facts.countries[0].takers).toEqual({pw: 3})
        expect(facts.countries[1].takers).toEqual({pw: 2, fr: 1})
    })

    it("lists each flag with what it took, filled, lost and from whom", () => {
        expect(facts.flags.map(({code, took, filled, lost, tookFrom}) => ({code, took, filled, lost, tookFrom}))).toEqual([
            {code: "pw", took: 5, filled: 0, lost: 0, tookFrom: {fr: 3, de: 2}},
            {code: "fr", took: 1, filled: 1, lost: 3, tookFrom: {de: 1}},
            {code: "de", took: 0, filled: 0, lost: 3, tookFrom: {}},
        ])
    })

    it("sums, for each flag, the tiles of the landmasses whose painted flag switched to it, by continent", () => {
        // The islet of Germany is too small to read from far.
        expect(facts.flags.map(({code, flippedByContinent}) => ({code, flippedByContinent}))).toEqual([
            {code: "pw", flippedByContinent: {Europe: 300}},
            {code: "fr", flippedByContinent: {Europe: 150}},
            {code: "de", flippedByContinent: {}},
        ])
    })

    it("judges nothing: no kind, no attacker, no place", () => {
        expect(Object.keys(facts).sort()).toEqual(["captured", "countries", "flags", "tilesChanged"])
    })
})

describe("the facts of a tour", () => {
    const changes = [change(1, "fr", "pw"), change(2, "fr", "pw"), change(11, "dz", "pw"), change(5, undefined, "pw"), change(7, "pw", "de")]
    const after = new Map<number, string>([[1, "pw"], [2, "pw"], [11, "pw"], [5, "pw"], [7, "de"]])
    const continentOf = (tile: number) => REGIONS[GROUND.get(tile) ?? ""]

    it("counts what the flag took on each continent and from whom", () => {
        const facts = tourFactsOf("pw", changes, after, continentOf, pointOf, [])

        expect(facts.took).toBe(3)
        expect(facts.byContinent).toEqual({Europe: 2, Africa: 1})
        expect(facts.tookFrom).toEqual({fr: 2, dz: 1})
        expect(facts.flippedByContinent).toEqual({})
    })

    it("sums the tiles of the landmasses whose painted flag switched to it, by continent: all a tour shows from far", () => {
        const flips = [
            {landmass: 1, tiles: 300, holders: ["fr", "pw"], tile: 1},
            {landmass: 2, tiles: 140, holders: ["de", "pw"], tile: 7},
            {landmass: 3, tiles: 80, holders: ["dz", "pw"], tile: 11},
            {landmass: 4, tiles: 500, holders: ["pw", "de"], tile: 2},
        ]

        const facts = tourFactsOf("pw", changes, after, continentOf, pointOf, flips)

        // The islet in Africa is too small to read.
        expect(facts.flippedByContinent).toEqual({Europe: 440})
    })
})

describe("the places held", () => {
    // A map of 1,200 tiles: a thousand of France, the rest of Germany and Algeria.
    const tiles = 1200
    const groundOf = (tile: number) => tile <= 1000 ? "fr" : tile <= 1100 ? "de" : "dz"
    const owners = new Map<number, string>()
    for (let tile = 1; tile <= 1200; tile++) owners.set(tile, tile <= 800 ? "pw" : tile <= 1100 ? "fr" : "dz")
    const regionOf = (country: string) => REGIONS[country]

    it("gives the top holder of the world, each continent and each big country, with its share and its rivals", () => {
        const fought = [change(1, "fr", "pw"), change(2, "fr", "pw"), change(1101, "fr", "dz")]

        const places = placesOf(owners, tiles, groundOf, fought, regionOf)

        expect(places.map(({kind, code, continent, tiles, holder, held, changed}) => ({kind, code, continent, tiles, holder, held, changed}))).toEqual([
            {kind: "world", code: undefined, continent: undefined, tiles: 1200, holder: "pw", held: 800, changed: 2},
            {kind: "continent", code: undefined, continent: "Europe", tiles: 1100, holder: "pw", held: 800, changed: 2},
            {kind: "country", code: "fr", continent: "Europe", tiles: 1000, holder: "pw", held: 800, changed: 2},
            {kind: "continent", code: undefined, continent: "Africa", tiles: 100, holder: "dz", held: 100, changed: 1},
        ])
        expect(places[0].share).toBeCloseTo(800 / 1200)
        expect(places[0].rivals).toEqual({fr: 300, dz: 100})
    })

    it("leaves out a place nothing changed hands in for its holder: there is nothing to play", () => {
        const places = placesOf(owners, tiles, groundOf, [change(1, "fr", "pw")], regionOf)

        expect(places.map(({kind, continent}) => `${kind} ${continent ?? ""}`)).toEqual(["world ", "continent Europe", "country Europe"])
    })

    it("tells a holding from a place: the holder, its share, the next holders as its victims", () => {
        const [world] = placesOf(owners, tiles, groundOf, [change(1, "fr", "pw")], regionOf)

        expect(holdingStoryOf(world)).toEqual({
            kind: "holding", attacker: "pw", rival: undefined, victims: ["fr", "dz"], place: {region: "the world"}, share: 800 / 1200,
        })
    })
})
