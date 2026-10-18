import {describe, expect, it} from "vitest"
import {TileChange} from "./changes.ts"
import {holdingsOf, inHolding} from "./holding.ts"

const REGIONS: Record<string, string> = {
    ru: "Europe", fr: "Europe", de: "Europe", hr: "Europe", pt: "Europe",
    br: "South America", ar: "South America", pw: "Oceania",
}
const regionOf = (country: string) => REGIONS[country]

// Each part: so many tiles of a country's ground, held by a flag (none for empty ground), in a row.
function mapOf(parts: [ground: string, count: number, owner: string | undefined][]) {
    const grounds: string[] = []
    const owners = new Map<number, string>()
    for (const [ground, count, owner] of parts) {
        for (let i = 0; i < count; i++) {
            grounds.push(ground)
            if (owner !== undefined) owners.set(grounds.length, owner)
        }
    }
    return {owners, tiles: grounds.length, groundOf: (tile: number) => grounds[tile - 1], first: (ground: string) => grounds.indexOf(ground) + 1}
}

function took(to: string, from: string | undefined, count: number, first: number): TileChange[] {
    return Array.from({length: count}, (_, i) => ({tile: first + i, from, to, at: i}))
}

describe("a holding", () => {
    it("is a continent most of which one flag holds, as the map stands, once it fought for it", () => {
        const map = mapOf([["fr", 400, "hr"], ["de", 100, "hr"], ["de", 100, "pt"], ["br", 1200, "br"], ["ar", 1200, "ar"]])

        const [best] = holdingsOf(map.owners, map.tiles, map.groundOf, took("hr", "fr", 60, 1), regionOf)

        expect(best).toEqual({
            story: {
                kind: "holding", attacker: "hr", rival: undefined, victims: ["pt"], place: {region: "Europe"}, taken: 500,
                share: 500 / 600,
            },
            ground: 600,
            short: undefined,
        })
    })

    it("counts empty ground in the place: a share is of all of it", () => {
        const map = mapOf([["fr", 300, "hr"], ["fr", 100, undefined], ["br", 1200, "br"], ["ar", 1200, "ar"]])

        const europe = holdingsOf(map.owners, map.tiles, map.groundOf, took("hr", "fr", 60, 1), regionOf)
            .find(({story}) => "region" in story.place && story.place.region === "Europe")

        expect(europe?.story.share).toBe(0.75)
    })

    it("counts Europe without Russia, which is a place of its own: holding Siberia is not holding Europe", () => {
        const map = mapOf([["ru", 1200, "pw"], ["fr", 400, "fr"], ["br", 1200, "br"], ["ar", 1200, "ar"]])

        const holdings = holdingsOf(map.owners, map.tiles, map.groundOf, took("pw", "ru", 60, 1), regionOf)

        expect(holdings.filter(({short}) => short === undefined).map(({story}) => story.place)).toEqual([{country: "ru"}])
        expect(inHolding({region: "Europe"}, "ru", regionOf)).toBe(false)
        expect(inHolding({region: "Europe"}, "fr", regionOf)).toBe(true)
    })

    it("is a big country most of which a foreign flag holds, never a country holding its own ground", () => {
        const map = mapOf([["br", 1000, "pt"], ["br", 200, "br"], ["ar", 1200, "ar"], ["fr", 400, "fr"]])

        const holdings = holdingsOf(map.owners, map.tiles, map.groundOf, [...took("pt", "br", 60, 1), ...took("ar", "fr", 60, 2401)], regionOf)

        expect(holdings.find(({story}) => story.attacker === "pt")).toMatchObject({story: {place: {country: "br"}}, short: undefined})
        expect(holdings.find(({story}) => story.attacker === "ar" && "country" in story.place)?.short).toBe("mostly its own country")
    })

    it("is no story when it is mostly the flag's own country, wherever it is counted", () => {
        const map = mapOf([["au", 1200, "au"], ["nz", 100, "pt"], ["fr", 400, "fr"]])

        const oceania = holdingsOf(map.owners, map.tiles, map.groundOf, took("au", "nz", 60, 1201), (country) =>
            country === "fr" ? "Europe" : "Oceania").find(({story}) => "region" in story.place && story.place.region === "Oceania")

        expect(oceania).toMatchObject({story: {attacker: "au", share: 1200 / 1300}, short: "mostly its own country"})
    })

    it("is no story when the flag holds too little of it, or fought for none of it in the window", () => {
        const map = mapOf([["br", 800, "pt"], ["br", 400, "br"], ["ar", 1200, "ar"], ["fr", 400, "hr"]])

        const holdings = holdingsOf(map.owners, map.tiles, map.groundOf, took("pt", "br", 60, 1), regionOf)

        expect(holdings.find(({story}) => "country" in story.place && story.place.country === "br")?.short).toBe("too little, 75% at least")
        expect(holdings.find(({story}) => "region" in story.place && story.place.region === "Europe")?.short)
            .toBe("it fought for none of it in the window")
    })

    it("is told once a flag, where it holds the most for the place's size: 80% of Europe over 13% of the world", () => {
        const elsewhere = Array.from({length: 10}, (_, i): [string, number, string] => [`c${i}`, 300, `c${i}`])
        const map = mapOf([["fr", 400, "hr"], ["de", 80, "hr"], ["de", 120, "pt"], ...elsewhere])

        const holdings = holdingsOf(map.owners, map.tiles, map.groundOf, took("hr", "fr", 60, 1), regionOf)
            .filter(({story}) => story.attacker === "hr")

        expect(holdings.map(({story, short}) => [story.place, short])).toEqual([
            [{region: "Europe"}, undefined],
            [{region: "the world"}, "told where it holds more"],
        ])
    })
})
