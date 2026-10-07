import {describe, expect, it} from "vitest"
import {TileChange} from "./changes.ts"
import {castOf, placeOf, Story, storyOf, THE_WORLD} from "./story.ts"

const REGIONS: Record<string, string> = {
    fr: "Europe",
    de: "Europe",
    gb: "Europe",
    es: "Europe",
    br: "South America",
}
const regionOf = (country: string) => REGIONS[country]

function took(to: string | undefined, from: string | undefined, count: number, first = 1): TileChange[] {
    return Array.from({length: count}, (_, i) => ({tile: first + i, from, to, at: i}))
}

describe("the story of a front", () => {
    it("is an attack on a region when the attacker took tiles over several countries", () => {
        const changes = [
            ...took("ps", "fr", 40, 1), ...took("ps", "de", 20, 41), ...took("ps", "gb", 20, 61), ...took("ps", "es", 20, 81),
        ]
        const ground = (tile: number) => tile <= 40 ? "fr" : tile <= 60 ? "de" : tile <= 80 ? "gb" : "es"

        expect(storyOf(changes, ground, regionOf)).toEqual({
            kind: "attack", attacker: "ps", rival: undefined, victims: ["fr", "de", "es"],
            place: {region: "Europe"}, taken: 100,
        })
    })

    it("is an invasion when nearly every tile it took is in one country", () => {
        const story = storyOf(took("dz", "fr", 90), () => "fr", regionOf)

        expect(story).toMatchObject({kind: "invasion", attacker: "dz", place: {country: "fr"}})
    })

    it("is a comeback when the attacker takes back its own ground", () => {
        const story = storyOf(took("ru", "ua", 90), () => "ru", regionOf)

        expect(story).toMatchObject({kind: "comeback", attacker: "ru", victims: ["ua"], place: {country: "ru"}})
    })

    it("is a comeback when a flag takes its own continent back from a flag from elsewhere", () => {
        const changes = [...took("be", "ps", 40, 1), ...took("be", "ps", 30, 41), ...took("be", "ps", 30, 71)]
        const ground = (tile: number) => tile <= 40 ? "fr" : tile <= 70 ? "gb" : "es"

        expect(storyOf(changes, ground, (country) => country === "ps" ? "Asia" : regionOf(country) ?? "Europe"))
            .toMatchObject({kind: "comeback", attacker: "be", place: {region: "Europe"}})
    })

    it("is an attack when a flag takes its own continent from a neighbour", () => {
        const changes = [...took("de", "fr", 40, 1), ...took("de", "fr", 30, 41), ...took("de", "fr", 30, 71)]
        const ground = (tile: number) => tile <= 40 ? "fr" : tile <= 70 ? "gb" : "es"

        expect(storyOf(changes, ground, regionOf)).toMatchObject({kind: "attack", place: {region: "Europe"}})
    })

    it("is a kickout when the attacker already held most of the country it takes", () => {
        const story = storyOf(took("fr", "pl", 90), () => "au", regionOf, undefined, () => "fr")

        expect(story).toMatchObject({kind: "kickout", attacker: "fr", victims: ["pl"], place: {country: "au"}})
    })

    it("is an invasion when somebody else held the country at the start", () => {
        expect(storyOf(took("fr", "pl", 90), () => "au", regionOf, undefined, () => "pl")).toMatchObject({kind: "invasion"})
    })

    it("is a battle when two flags took about as much", () => {
        const changes = [...took("fr", "ps", 50, 1), ...took("ps", "fr", 40, 51)]

        expect(storyOf(changes, () => "fr", regionOf)).toMatchObject({kind: "battle", attacker: "fr", rival: "ps"})
    })

    it("follows the attacker it is told", () => {
        const changes = [...took("fr", "ps", 50, 1), ...took("de", "ps", 5, 51)]

        expect(storyOf(changes, () => "fr", regionOf, "de")).toMatchObject({attacker: "de", taken: 5})
    })

    it("is nothing when nobody took a tile", () => {
        expect(storyOf(took(undefined, "fr", 10), () => "fr", regionOf)).toBeUndefined()
    })
})

describe("the place of a story", () => {
    it("is the continent most of it happened in", () => {
        expect(placeOf(["fr", "fr", "de", "de", "fr", "de", "fr", "gb"], regionOf)).toEqual({region: "Europe"})
    })

    it("is the two countries most of it happened in, when no continent holds most of it", () => {
        expect(placeOf(["eg", "eg", "eg", "tr", "tr", "tr", "rs"], (country) => ({eg: "Africa", tr: "Asia", rs: "Europe"})[country]))
            .toEqual({countries: ["eg", "tr"]})
    })

    it("is the world when nothing holds most of it", () => {
        expect(placeOf(["fr", "br", "jp", "au"], regionOf)).toEqual({region: THE_WORLD})
        expect(placeOf([undefined], regionOf)).toEqual({region: THE_WORLD})
    })
})

describe("who a story is about", () => {
    const continents: Record<string, string> = {be: "Europe", de: "Europe", pl: "Europe", fr: "Europe", ps: "Asia", il: "Asia"}
    const continentOf = (country: string) => continents[country]
    const comeback = (attacker: string): Story =>
        ({kind: "comeback", attacker, rival: undefined, victims: ["ps"], place: {region: "Europe"}, taken: 10})

    it("is its flag when that flag leads what is taken around it", () => {
        const around = [...took("be", "ps", 60), ...took("de", "ps", 20, 61), ...took("il", "ps", 20, 81)]

        expect(castOf(comeback("be"), around, continentOf)).toEqual(comeback("be"))
    })

    it("is its continent when the continent's flags strike back together and none of them leads", () => {
        const around = [...took("be", "ps", 30), ...took("de", "ps", 25, 31), ...took("pl", "ps", 25, 56), ...took("il", "ps", 20, 81)]

        expect(castOf(comeback("de"), around, continentOf)).toMatchObject({attacker: "de", team: "Europe"})
    })

    it("is nobody when a minor flag is not part of a continent striking back", () => {
        const around = [...took("il", "ps", 70), ...took("de", "ps", 10, 71), ...took("fr", "ps", 20, 81)]

        expect(castOf(comeback("de"), around, continentOf)).toBeUndefined()
    })
})
