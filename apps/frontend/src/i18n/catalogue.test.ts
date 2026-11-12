import {describe, expect, it} from "vitest"
import {LANGUAGES, SOURCE_LANGUAGE} from "./languages.ts"
import {parseMessage, pluralsOf, Slot, slotsOf} from "./message.ts"
import {ENGLISH, ENGLISH_LANDMASSES, OWN_COUNTRY_NAMES} from "./translator.ts"
import NOTES from "./notes.json"
import FLAGS from "../../static/countries/countries.json"

const FILES = import.meta.glob<Record<string, unknown>>("./locales/*.json", {eager: true, import: "default"})
const LANDMASS_FILES = import.meta.glob<Record<string, unknown>>("./landmasses/*.json", {eager: true, import: "default"})
const COUNTRY_FILES = import.meta.glob<Record<string, unknown>>("./countries/*.json", {eager: true, import: "default"})

const linesOf = (language: string) => FILES[`./locales/${language}.json`]

// A count may be a plain number where a language has one form for every count.
const kindOf = (slot: Slot) => slot === "count" ? "value" : slot

describe("the catalogue", () => {
    it("has a file for each language and none for another", () => {
        expect(Object.keys(FILES).sort()).toEqual(LANGUAGES.map((language) => `./locales/${language}.json`).sort())
    })

    it("holds notes only for lines English has", () => {
        expect(Object.keys(NOTES).filter((key) => !(key in ENGLISH))).toEqual([])
    })

    it("names in English each flag it names itself", () => {
        for (const code of OWN_COUNTRY_NAMES) expect(ENGLISH).toHaveProperty(`country.${code}`)
    })

    for (const language of LANGUAGES) {
        describe(language, () => {
            const lines = linesOf(language)
            const cardinals = new Intl.PluralRules(language).resolvedOptions().pluralCategories
            const ordinals = new Intl.PluralRules(language, {type: "ordinal"}).resolvedOptions().pluralCategories

            if (language !== SOURCE_LANGUAGE) it("names every landmass English names, and no other", () => {
                const names = LANDMASS_FILES[`./landmasses/${language}.json`] ?? {}
                expect(Object.keys(names).filter((id) => !(id in ENGLISH_LANDMASSES)), "ids English no longer has").toEqual([])
                expect(Object.keys(ENGLISH_LANDMASSES).filter((id) => typeof names[id] !== "string" || !names[id]), "names missing").toEqual([])
            })

            it("says \"in\" each flag's country, with the flag's picture in it, and no other", () => {
                const phrases = COUNTRY_FILES[`./countries/${language}.json`] ?? {}
                const keys = Object.keys(FLAGS).map((code) => `in.${code}`)
                expect(Object.keys(phrases).filter((key) => !keys.includes(key)), "flags the game no longer has").toEqual([])
                const problems = keys.filter((key) => {
                    const phrase = phrases[key]
                    if (typeof phrase !== "string") return true
                    try {
                        return [...slotsOf(parseMessage(phrase)).keys()].join() !== "flag"
                    } catch {
                        return true
                    }
                })
                expect(problems, "missing, or not one {flag}").toEqual([])
            })

            it("holds every English key and no other", () => {
                const keys = Object.keys(lines)
                expect(keys.filter((key) => !(key in ENGLISH)), "keys English no longer has").toEqual([])
                expect(Object.keys(ENGLISH).filter((key) => !(key in lines)), "keys missing").toEqual([])
            })

            it("holds lines that read, with English's slots and every plural form the language needs", () => {
                const problems: string[] = []
                for (const [key, english] of Object.entries(ENGLISH)) {
                    const line = lines[key]
                    if (typeof line !== "string") {
                        problems.push(`${key}: not a line`)
                        continue
                    }
                    let nodes
                    try {
                        nodes = parseMessage(line)
                    } catch (e) {
                        problems.push(`${key}: ${(e as Error).message}`)
                        continue
                    }
                    if (language === SOURCE_LANGUAGE) parseMessage(english)
                    const want = [...slotsOf(parseMessage(english))].map(([name, slot]) => `${name}:${kindOf(slot)}`).sort()
                    const have = [...slotsOf(nodes)].map(([name, slot]) => `${name}:${kindOf(slot)}`).sort()
                    if (want.join() !== have.join()) problems.push(`${key}: slots ${have.join(" ")} where English has ${want.join(" ")}`)
                    for (const plural of pluralsOf(nodes)) {
                        const needed = plural.ordinal ? ordinals : cardinals
                        const missing = needed.filter((category) => !plural.selectors.includes(category))
                        if (missing.length) problems.push(`${key}: {${plural.name}} lacks ${missing.join(", ")}`)
                    }
                }
                expect(problems).toEqual([])
            })
        })
    }
})
