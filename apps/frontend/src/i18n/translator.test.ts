import {describe, expect, it} from "vitest"
import {Key, Translator} from "./translator.ts"
import {MessageError, parseMessage} from "./message.ts"
import {chooseLanguage, languageOf, nativeNameOf} from "./languages.ts"
import {localeOf} from "./load.ts"

const speaking = (language: "en" | "pl" | "fr" | "id", lines: Record<string, string>, locale?: string) =>
    new Translator(language, lines, locale)

const key = (name: string) => name as Key

describe("a line", () => {
    it("fills its slots, numbers in the language's digits", () => {
        const fr = speaking("fr", {[key("a")]: "{who} a pris {count} cases"})

        expect(fr.t(key("a"), {who: "Léa", count: 12345})).toBe("Léa a pris 12 345 cases")
    })

    it("picks each language's plural form, Polish's few and many among them", () => {
        const pl = speaking("pl", {[key("a")]: "{count, plural, one {# pole} few {# pola} many {# pól} other {# pola}}"})
        const said = (count: number) => pl.t(key("a"), {count})

        expect([1, 2, 5, 22, 25].map(said)).toEqual(["1 pole", "2 pola", "5 pól", "22 pola", "25 pól"])
    })

    it("takes an exact count before the category", () => {
        const en = speaking("en", {[key("a")]: "{count, plural, =0 {none} one {one tile} other {# tiles}}"})

        expect([0, 1, 4].map((count) => en.t(key("a"), {count}))).toEqual(["none", "one tile", "4 tiles"])
    })

    it("says an ordinal by the language's ordinal rules", () => {
        const en = speaking("en", {[key("format.ordinal")]: "{rank, selectordinal, one {#st} two {#nd} few {#rd} other {#th}}"})

        expect([1, 2, 3, 4, 11, 12, 13, 21, 102].map((rank) => en.ordinal(rank)))
            .toEqual(["1st", "2nd", "3rd", "4th", "11th", "12th", "13th", "21st", "102nd"])
    })

    it("keeps an element in its slot and hands a tag the words it wraps", () => {
        const en = speaking("en", {[key("a")]: "{winner} knocked out {loser}, <link>see how</link>"})

        const parts = en.parts<{b: string}>(key("a"), {
            winner: {b: "France"},
            loser: {b: "Spain"},
            link: (children) => ({b: `[${children.join("")}]`}),
        })

        expect(parts).toEqual([{b: "France"}, " knocked out ", {b: "Spain"}, ", ", {b: "[see how]"}])
    })

    it("falls back to English for a line its file lacks", () => {
        const fr = speaking("fr", {})

        expect(fr.t("country.xb")).toBe("Brittany")
    })

    it("refuses a line that does not read", () => {
        expect(() => parseMessage("{count, plural, one {# tile}}")).toThrow(MessageError)
        expect(() => parseMessage("<b>bold")).toThrow(MessageError)
        expect(() => parseMessage("{count")).toThrow(MessageError)
    })

    it("asks for every value it shows", () => {
        const en = speaking("en", {[key("a")]: "{who} won"})

        expect(() => en.t(key("a"))).toThrow("no value for {who}")
    })
})

describe("a country's name", () => {
    const names = (language: "en" | "fr" | "pl", codes: string[]) => codes.map((code) => speaking(language, {}).country(code))

    it("is the browser's, short when that is not an abbreviation", () => {
        expect(names("en", ["fr", "gb", "us", "hk", "mm", "ba"]))
            .toEqual(["France", "United Kingdom", "United States", "Hong Kong", "Myanmar", "Bosnia"])
        expect(names("fr", ["gb", "de"])).toEqual(["Royaume-Uni", "Allemagne"])
    })

    it("is the game's own line for a flag ISO has no code for, or one CLDR names as two places", () => {
        expect(names("en", ["xb", "gb-sct", "cd", "cg", "sx"])).toEqual(["Brittany", "Scotland", "DR Congo", "Congo", "Sint Maarten"])
    })
})

describe("the clock", () => {
    const en = speaking("en", {})

    it("says a countdown from its largest unit", () => {
        expect(en.countdown({days: 2, hours: 5, minutes: 3, seconds: 9})).toBe("2d 05h 03m 09s")
        expect(en.countdown({days: 0, hours: 5, minutes: 3, seconds: 9})).toBe("5h 03m 09s")
        expect(en.countdown({days: 0, hours: 0, minutes: 3, seconds: 9})).toBe("03m 09s")
        expect(en.countdown({days: 0, hours: 0, minutes: 0, seconds: 0})).toBe("00m 00s")
        expect(en.countdown({days: 2, hours: 5, minutes: 3, seconds: 9}, 2)).toBe("2d 05h")
    })

    it("says how long ago, in words for the last moments", () => {
        const now = 1_000_000_000
        expect(en.ago(now - 10_000, now)).toBe("now")
        expect(en.ago(now - 3 * 3_600_000, now)).toBe("3 hr. ago")
    })

    it("says a length in hours when it is whole hours", () => {
        expect([60, 120, 90].map((minutes) => en.length(minutes))).toEqual(["1 hour", "2 hours", "90 minutes"])
    })
})

describe("the language", () => {
    it("is the player's pick first, then CrazyGames', then the browser's first one the game speaks", () => {
        expect(chooseLanguage({picked: "de", portal: "fr-FR", browser: ["pl"]})).toBe("de")
        expect(chooseLanguage({picked: null, portal: "fr-FR", browser: ["pl"]})).toBe("fr")
        expect(chooseLanguage({picked: "xx", browser: ["ru-RU", "pl-PL", "en"]})).toBe("pl")
        expect(chooseLanguage({picked: null, browser: ["ru", "ja"]})).toBe("en")
    })

    it("reads every Portuguese as Brazil's, every Spanish as one, and Slovak as Czech", () => {
        expect(["pt-PT", "pt", "es-MX", "es-419", "sk-SK", "in", "en-AU"].map(languageOf))
            .toEqual(["pt-BR", "pt-BR", "es", "es", "cs", "id", "en"])
    })

    it("is formatted in the browser's own variant of it", () => {
        expect(localeOf("en", ["fr-FR", "en-GB"])).toBe("en-GB")
        expect(localeOf("cs", ["sk-SK"])).toBe("cs")
    })

    it("is named in itself", () => {
        expect(nativeNameOf("de")).toBe("Deutsch")
        expect(nativeNameOf("cs")).toBe("Čeština")
    })
})

describe("a landmass's name", () => {
    it("is its own, or for a country's main one the country's", () => {
        const en = new Translator("en", {})

        expect(en.landmass(264, "fr")).toBe("Corsica")
        expect(en.landmass(231, "fr")).toBe("France")
    })

    it("is English where the language's file has none yet", () => {
        const fr = new Translator("fr", {}, "fr", {"264": "Corse"})

        expect(fr.landmass(264, "fr")).toBe("Corse")
        expect(fr.landmass(13, "hr")).toBe("Split-Dalmatia")
    })
})

describe("a region's name", () => {
    it("is its line", () => {
        expect(new Translator("en", {}).region("Middle East")).toBe("Middle East")
    })
})
