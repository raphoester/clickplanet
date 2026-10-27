import {describe, expect, it} from "vitest"
import {Day, Story} from "../domain/clip/story.ts"
import {voiceOf} from "./callUp.ts"
import {wordsOf} from "./overlay.ts"

const CUTOFF = Date.parse("2026-10-23T21:00:00Z")
const FINALE = Date.parse("2026-10-31T21:00:00Z")

const day: Day = {
    season: 0, number: 15, finale: false, own: false, endedAt: CUTOFF,
    podium: [{country: "dz", rank: 1, points: 25}],
    table: [
        {country: "dz", rank: 1, points: 289}, {country: "pl", rank: 2, points: 275}, {country: "br", rank: 3, points: 145},
        {country: "fr", rank: 4, points: 117}, {country: "gb", rank: 5, points: 100}, {country: "ro", rank: 6, points: 90},
        {country: "tr", rank: 7, points: 80},
    ],
}

function calledUp(country: string, table = day.table): Story {
    const ranked = [...table].sort((a, b) => b.points - a.points)
    const against = ranked[0].country === country ? ranked[1] : ranked[0]
    return {
        kind: "callup", attacker: country, rival: undefined, victims: [against.country], place: {region: "the world"},
        day: {...day, table}, finaleAt: FINALE,
    }
}

describe("a country called up to the Final Battle", () => {
    it("is told in its own language: how far behind the leader, and when the battle starts where it lives", () => {
        const words = wordsOf(calledUp("pl"))

        expect(words.headline).toBe("POLSKA: 14 PUNKTÓW STRATY")
        expect(words.line).toBe("FINAŁOWA BITWA: SOBOTA, 31 PAŹDZIERNIKA 22:00")
        expect(words.call).toBe("TWÓJ KRAJ CIĘ POTRZEBUJE")
        expect(words.callFlags).toEqual(["pl"])
        expect(words.link).toBe("https://clickplanet.lol/?f=pl")
        expect(words.table).toEqual([
            {rank: 1, country: "dz", name: "ALGIERIA", points: "289"}, {rank: 2, country: "pl", name: "POLSKA", points: "275"},
        ])
    })

    it("says the leader's lead over the next, in English when the clip has no words in its language", () => {
        const words = wordsOf(calledUp("dz"))

        expect(words.headline).toBe("ALGERIA LEADS THE SEASON BY 14 POINTS")
        expect(words.line).toBe("FINAL BATTLE: SATURDAY 31 OCTOBER AT 22:00")
        expect(words.call).toBe("YOUR COUNTRY NEEDS YOU")
        expect(words.table?.map(({country}) => country)).toEqual(["dz", "pl"])
    })

    it("names an English-speaking country with its article and the leader it chases", () => {
        expect(wordsOf(calledUp("gb")).headline).toBe("THE UK IS 189 POINTS BEHIND ALGERIA")
    })

    it("keeps French typography", () => {
        expect(wordsOf(calledUp("fr")).headline).toBe("FRANCE\u00a0: 172 POINTS DE RETARD")
    })

    it("counts points as each language does", () => {
        const one = [{country: "dz", rank: 1, points: 289}, {country: "pl", rank: 2, points: 288}]
        const few = [{country: "dz", rank: 1, points: 289}, {country: "pl", rank: 2, points: 267}]
        const twenty = [{country: "dz", rank: 1, points: 289}, {country: "ro", rank: 2, points: 269}]

        expect(wordsOf(calledUp("pl", one)).headline).toBe("POLSKA: 1 PUNKT STRATY")
        expect(wordsOf(calledUp("pl", few)).headline).toBe("POLSKA: 22 PUNKTY STRATY")
        expect(wordsOf(calledUp("ro", twenty)).headline).toBe("ROMÂNIA: 20 DE PUNCTE ÎN URMĂ")
        expect(wordsOf(calledUp("tr")).headline).toBe("TÜRKİYE: 209 PUAN GERİDE")
    })

    it("says two countries level at the top are tied", () => {
        const level = [{country: "dz", rank: 1, points: 289}, {country: "pl", rank: 1, points: 289}]

        expect(wordsOf(calledUp("pl", level)).headline).toBe("POLSKA: REMIS NA SZCZYCIE")
        expect(wordsOf(calledUp("dz", level)).headline).toBe("ALGERIA IS TIED FOR FIRST")
    })

    it("names the zone of its clock when the country's zones read different times, and speaks to Brazil as Brazil does", () => {
        const words = wordsOf(calledUp("br"))

        expect(words.line).toBe("BATALHA FINAL: SÁBADO, 31 DE OUTUBRO ÀS 18:00 BRT")
        expect(words.call).toBe("SEU PAÍS PRECISA DE VOCÊ")
    })

    it("counts tiles and reads the clock in its own language and zone", () => {
        const speech = wordsOf(calledUp("pl")).speech

        expect(speech?.tiles(1)).toBe("1 pole")
        expect(speech?.tiles(3)).toBe("3 pola")
        expect(speech?.tiles(12_345)).toBe("12\u00a0345 pól")
        expect(speech?.clock(CUTOFF)).toBe("pt., 23 paź, 23:00 CEST")
    })

    it("never gives the headline back as a teaser", () => {
        for (const country of ["dz", "pl", "br", "fr", "gb", "ro", "tr"]) {
            const words = wordsOf(calledUp(country))
            expect(words.teasers.length).toBeGreaterThanOrEqual(2)
            expect(words.teasers).not.toContain(words.headline)
            expect(words.teaserNames).toHaveLength(words.teasers.length)
        }
    })

    it("names its teasers the same in every language", () => {
        expect(wordsOf(calledUp("pl")).teaserNames).toEqual(["can-still-win-the-season", "gap-to-first-place"])
        expect(wordsOf(calledUp("gb")).teaserNames).toEqual(wordsOf(calledUp("pl")).teaserNames)
        expect(wordsOf(calledUp("dz")).teaserNames).toEqual(["can-anyone-stop", "clear-at-the-top"])
    })
})

describe("the voice of a country", () => {
    it("is the language CLDR gives the country, else English, and the time most of its zones read", () => {
        expect(voiceOf("pl", FINALE)).toEqual({language: "pl", locale: "pl-PL", region: "PL", timeZone: "Europe/Warsaw", zoneNamed: false})
        expect(voiceOf("dz", FINALE)).toEqual({language: "en", locale: "en-GB", region: "DZ", timeZone: "Africa/Algiers", zoneNamed: false})
        expect(voiceOf("pt", FINALE)).toMatchObject({language: "pt", timeZone: "Atlantic/Madeira", zoneNamed: true})
        expect(voiceOf("us", FINALE)).toMatchObject({language: "en", locale: "en-US", zoneNamed: true})
    })
})
