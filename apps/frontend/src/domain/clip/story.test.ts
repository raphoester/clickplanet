import {describe, expect, it} from "vitest"
import {regionOf} from "../regions.ts"
import {fightersOf, inPlace, sidesOf, Story, storyCountriesOf, storyKeyOf} from "./story.ts"

const attack: Story = {kind: "attack", attacker: "hr", rival: undefined, victims: ["fr"], place: {region: "Europe"}}

describe("the sides of a story", () => {
    it("are the attacker, and whoever it fights or kicks out", () => {
        expect(sidesOf(attack)).toEqual(["hr"])
        expect(sidesOf({...attack, kind: "battle", rival: "fr"})).toEqual(["hr", "fr"])
        expect(sidesOf({...attack, kind: "kickout", place: {country: "de"}})).toEqual(["hr", "fr"])
    })

    it("are the loser first for a flag thrown out, against the continent's flag when its flags threw it out together", () => {
        expect(sidesOf({...attack, kind: "rout"})).toEqual(["fr"])
        expect(sidesOf({...attack, kind: "rout", team: "Europe"})).toEqual(["fr", "eu"])
    })

    it("are the continent's own flag when its flags strike back together, or its leading flag's with none", () => {
        expect(sidesOf({...attack, kind: "comeback", team: "Europe"})).toEqual(["eu"])
        expect(sidesOf({...attack, kind: "comeback", attacker: "dz", place: {region: "Africa"}, team: "Africa"})).toEqual(["dz"])
    })
})

describe("the key of a story", () => {
    it("is its first side on the continent it is told on, and what it tells of it there", () => {
        expect(storyKeyOf(attack, regionOf)).toEqual({flag: "hr", continent: "Europe"})
        expect(storyKeyOf({...attack, place: {country: "dz"}}, regionOf)).toEqual({flag: "hr", continent: "Africa"})
        expect(storyKeyOf({...attack, kind: "rout", place: {countries: ["eg", "tr"]}}, regionOf)).toEqual({flag: "fr", continent: "Africa"})
        expect(storyKeyOf({...attack, place: {region: "the world"}}, regionOf)).toEqual({flag: "hr", continent: "the world"})
        expect(storyKeyOf({...attack, kind: "holding", victims: ["se"], share: 0.8}, regionOf)).toEqual({flag: "hr", continent: "Europe", angle: "holding"})
    })

    it("names the day a round tells or a call-up follows", () => {
        const day = {season: 1, number: 5, finale: false, endedAt: 0, podium: [], table: []}

        expect(storyKeyOf({...attack, kind: "round", day}, regionOf)).toEqual({flag: "hr", continent: "Europe", angle: "round", round: "1.5"})
        expect(storyKeyOf({...attack, kind: "callup", day}, regionOf)).toEqual({flag: "hr", continent: "Europe", angle: "callup", round: "1.5"})
    })
})

describe("the countries a story shows", () => {
    it("are its place's, its sides and the flags it takes from, only the game's", () => {
        expect(storyCountriesOf({...attack, kind: "battle", rival: "de", place: {countries: ["eg", "tr"]}})).toEqual(["eg", "tr", "hr", "de", "fr"])
        expect(storyCountriesOf({...attack, kind: "comeback", team: "Europe"})).toEqual(["hr", "fr"])
    })
})

describe("who fights on a story's side", () => {
    it("is its attacker, and its rival in a battle", () => {
        expect(["hr", "fr", "de"].filter(fightersOf({...attack, kind: "battle", rival: "fr"}, regionOf))).toEqual(["hr", "fr"])
        expect(["hr", "fr", "de"].filter(fightersOf(attack, regionOf))).toEqual(["hr"])
    })

    it("is the loser of a rout, whoever takes its land, and every flag of a continent striking back together", () => {
        expect(["hr", "fr", "de"].filter(fightersOf({...attack, kind: "rout"}, regionOf))).toEqual(["fr"])
        expect(["hr", "fr", "dz", undefined].filter(fightersOf({...attack, kind: "comeback", team: "Europe"}, regionOf))).toEqual(["hr", "fr"])
    })
})

describe("a country's ground in a place", () => {
    it("is the place's country or one of its two, any of its continent, and anything in the world", () => {
        expect(inPlace({country: "fr"}, "fr", regionOf)).toBe(true)
        expect(inPlace({country: "fr"}, "de", regionOf)).toBe(false)
        expect(inPlace({countries: ["eg", "tr"]}, "tr", regionOf)).toBe(true)
        expect(inPlace({region: "Europe"}, "de", regionOf)).toBe(true)
        expect(inPlace({region: "Europe"}, "dz", regionOf)).toBe(false)
        expect(inPlace({region: "the world"}, undefined, regionOf)).toBe(true)
        expect(inPlace({country: "fr"}, undefined, regionOf)).toBe(false)
    })
})
