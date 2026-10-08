import {describe, expect, it} from "vitest"
import {TileChange} from "./changes.ts"
import {castOf, inPlace, placeOf, routOf, sameRout, sameStory, Story, storyOf, THE_WORLD, widerThan} from "./story.ts"

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

    it("is an attack when a flag takes its own continent from its neighbours more than from flags from elsewhere", () => {
        const changes = [...took("pt", "il", 40, 1), ...took("pt", "de", 30, 41), ...took("pt", "es", 30, 71)]
        const ground = (tile: number) => tile <= 40 ? "fr" : tile <= 70 ? "de" : "es"

        expect(storyOf(changes, ground, (country) => country === "il" ? "Asia" : "Europe"))
            .toMatchObject({kind: "attack", attacker: "pt", victims: ["il", "de", "es"], place: {region: "Europe"}})
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

    it("is no battle when two flags took about as much from a third: they are allies", () => {
        const changes = [...took("il", "ps", 50, 1), ...took("be", "ps", 40, 51)]

        expect(storyOf(changes, () => "fr", regionOf)).toMatchObject({rival: undefined, attacker: "il"})
    })

    it("is a battle when two flags took about as much", () => {
        const changes = [...took("fr", "ps", 50, 1), ...took("ps", "fr", 40, 51)]

        expect(storyOf(changes, () => "fr", regionOf)).toMatchObject({kind: "battle", attacker: "fr", rival: "ps"})
    })

    it("is no battle when the other flag is busy taking a third and only the attacker fights it", () => {
        const changes = [...took("pt", "pl", 100, 1), ...took("pt", "ro", 40, 101), ...took("ro", "dz", 300, 141), ...took("ro", "pt", 1, 441)]

        expect(storyOf(changes, () => "us", regionOf, "pt")).toMatchObject({rival: undefined, attacker: "pt"})
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

describe("a flag thrown out", () => {
    const europe: Story = {
        kind: "comeback", attacker: "de", rival: undefined, victims: ["ps"], place: {region: "Europe"}, taken: 10, team: "Europe",
    }

    it("is the story when a continent striking back takes half of what it held there", () => {
        expect(routOf(europe, {before: 1000, after: 400})).toMatchObject({kind: "rout", victims: ["ps"], team: "Europe"})
    })

    it("is not when it kept more than half", () => {
        expect(routOf(europe, {before: 1000, after: 700})).toEqual(europe)
    })

    it("is not when it only had a foothold there", () => {
        expect(routOf(europe, {before: 100, after: 0})).toEqual(europe)
    })

    it("is one attacker kicking it out when one flag attacks and takes half of what it held there", () => {
        const israel = {...europe, kind: "attack" as const, attacker: "il", team: undefined}

        expect(routOf(israel, {before: 1000, after: 300})).toMatchObject({kind: "kickout", attacker: "il", victims: ["ps"]})
    })

    it("is one rout however many sides tell it", () => {
        const thrownOut = routOf(europe, {before: 1000, after: 0})
        const kicked = routOf({...europe, kind: "attack", attacker: "il", team: undefined}, {before: 1000, after: 0})
        const name = (place: Story["place"]) => "region" in place ? place.region : "elsewhere"

        expect(sameRout(thrownOut, kicked, name)).toBe(true)
        expect(sameRout(thrownOut, {...kicked, victims: ["bg"]}, name)).toBe(false)
    })

    it("is the flag losing it when many flags take it and none of them leads", () => {
        const many = {...europe, kind: "attack" as const, attacker: "cy", team: undefined}

        expect(routOf(many, {before: 1000, after: 200}, false)).toMatchObject({kind: "rout", victims: ["ps"]})
    })

    it("is never one attacker kicking a flag out of its own land", () => {
        const morocco = {...europe, kind: "invasion" as const, attacker: "ma", victims: ["es"], place: {country: "es"}, team: undefined}

        expect(routOf(morocco, {before: 1000, after: 200}, true, true)).toEqual(morocco)
    })

    it("is not when one flag leads the comeback: that flag is the story", () => {
        const belgium = {...europe, attacker: "be", team: undefined}

        expect(routOf(belgium, {before: 1000, after: 0})).toEqual(belgium)
    })
})

describe("the ground of a place", () => {
    const continentOf = (country: string) => ({fr: "Europe", tr: "Asia"})[country]

    it("is a country's, two countries', or a continent's", () => {
        expect(inPlace({country: "fr"}, "fr", continentOf)).toBe(true)
        expect(inPlace({countries: ["eg", "tr"]}, "tr", continentOf)).toBe(true)
        expect(inPlace({region: "Europe"}, "fr", continentOf)).toBe(true)
        expect(inPlace({region: "Europe"}, "tr", continentOf)).toBe(false)
        expect(inPlace({region: "Europe"}, undefined, continentOf)).toBe(false)
    })
})

describe("the same story", () => {
    const name = (place: Story["place"]) => "country" in place ? place.country : "region" in place ? place.region : "both"
    const continents: Record<string, string> = {fr: "Europe", es: "Europe", de: "Europe", ma: "Africa", sd: "Africa", in: "Asia"}
    const continentOf = (country: string) => continents[country]
    const same = (a: Story, b: Story) => sameStory(a, b, name, continentOf)
    const italy = (place: Story["place"]): Story => ({kind: "attack", attacker: "it", rival: undefined, victims: ["de"], place, taken: 10})

    it("is one flag beating another, at any scale", () => {
        expect(same(italy({region: "Europe"}), {...italy({country: "fr"}), kind: "kickout"})).toBe(true)
        expect(same(italy({countries: ["fr", "es"]}), italy({country: "es"}))).toBe(true)
    })

    it("is one flag taking a continent and a country of it, whoever it beat there", () => {
        expect(same(italy({region: "Europe"}), {...italy({country: "es"}), kind: "invasion", victims: ["ma"]})).toBe(true)
    })

    it("is not the same flag beating another flag elsewhere", () => {
        expect(same(italy({country: "fr"}), {...italy({country: "es"}), victims: ["ma"]})).toBe(false)
    })

    it("is a flag's tour of the world beside its stories on each continent", () => {
        expect(same(italy({region: THE_WORLD}), italy({region: "Europe"}))).toBe(false)
        expect(same(italy({region: THE_WORLD}), {...italy({region: THE_WORLD}), victims: ["ma"]})).toBe(true)
    })

    it("is told over the most of the map", () => {
        expect(widerThan({region: "Europe"}, {country: "de"})).toBe(true)
        expect(widerThan({region: THE_WORLD}, {region: "Europe"})).toBe(true)
        expect(widerThan({country: "de"}, {countries: ["de", "pl"]})).toBe(false)
        expect(widerThan({region: "Asia"}, {region: "Europe"})).toBe(false)
    })

    it("is not the same flag beating one flag on two continents", () => {
        expect(same(italy({region: "Africa"}), {...italy({country: "in"}), kind: "kickout"})).toBe(false)
        expect(same(italy({country: "fr"}), italy({country: "es"}))).toBe(false)
    })

    it("is one flag striking back, wherever in one window", () => {
        const back = (place: Story["place"], victim: string): Story => ({...italy(place), kind: "comeback", attacker: "ma", victims: [victim]})

        expect(same(back({country: "ma"}, "gb"), back({region: "Africa"}, "il"))).toBe(true)
    })

    it("is not a flag losing a continent and its top taker beating it in one country", () => {
        const losing: Story = {...italy({region: "Africa"}), kind: "rout", attacker: "il", victims: ["dz"]}

        expect(same(losing, {...italy({country: "sd"}), kind: "kickout", attacker: "il", victims: ["dz"]})).toBe(false)
    })

    it("is a battle only when both sides fight over the same place", () => {
        const battle = (place: Story["place"]): Story => ({...italy(place), kind: "battle", rival: "de"})

        expect(same(battle({country: "de"}), battle({country: "de"}))).toBe(true)
        expect(same(battle({country: "de"}), italy({region: "Europe"}))).toBe(false)
    })
})
