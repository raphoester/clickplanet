import {readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {LANGUAGES, SOURCE_LANGUAGE} from "./languages.ts"
import {parseMessage, pluralsOf, Slot, slotsOf} from "./message.ts"
import {ENGLISH, OWN_COUNTRY_NAMES} from "./translator.ts"
import NOTES from "./notes.json"
import GLOSSARY from "./glossary.json"
import FLAGS from "../../static/countries/countries.json"
import {BOARDS} from "../app/viewer/boardsAsset.ts"

const FILES = import.meta.glob<Record<string, unknown>>("./locales/*.json", {eager: true, import: "default"})
const LANDMASS_FILES = import.meta.glob<Record<string, unknown>>("./landmasses/*/*.json", {eager: true, import: "default"})
const PROVINCE_FILES = import.meta.glob<Record<string, unknown>>("./provinces/*/*.json", {eager: true, import: "default"})
const COUNTRY_FILES = import.meta.glob<Record<string, unknown>>("./countries/*.json", {eager: true, import: "default"})

const linesOf = (language: string) => FILES[`./locales/${language}.json`]

const STATIC = fileURLToPath(new URL("../../static/", import.meta.url))

// Each board's landmass names in English, as the page fetches them beside the board.
const ENGLISH_LANDMASSES = Object.fromEntries(Object.values(BOARDS).map((board) => [
    board.name,
    (JSON.parse(readFileSync(`${STATIC}boards/${board.name}/${board.page.landmassNames}`, "utf8")) as {names: Record<string, string>}).names,
]))

// Each board's province names in English, for the boards that have provinces (docs/provinces.md).
const ENGLISH_PROVINCES = Object.fromEntries(Object.values(BOARDS).filter((board) => board.page.provinceNames).map((board) => [
    board.name,
    (JSON.parse(readFileSync(`${STATIC}boards/${board.name}/${board.page.provinceNames}`, "utf8")) as {names: Record<string, string>}).names,
]))

// Boards whose landmass names are not translated yet, read in English (load.ts): none since PR 5 wrote siege-1's, after
// earth-2's (2026-11-18). A new board waits here until its names are written.
const AWAITING_TRANSLATION = new Set<string>()

// A count may be a plain number where a language has one form for every count.
const kindOf = (slot: Slot) => slot === "count" ? "value" : slot

describe("the catalogue", () => {
    it("has a file for each language and none for another", () => {
        expect(Object.keys(FILES).sort()).toEqual(LANGUAGES.map((language) => `./locales/${language}.json`).sort())
    })

    it("names the landmasses of the boards the page carries, and of no other", () => {
        const boards = new Set(Object.keys(LANDMASS_FILES).map((path) => path.split("/")[2]))
        expect([...boards].filter((board) => !(board in BOARDS))).toEqual([])
    })

    it("names the provinces of the boards that have them, and of no other", () => {
        const boards = new Set(Object.keys(PROVINCE_FILES).map((path) => path.split("/")[2]))
        expect([...boards].filter((board) => !(board in ENGLISH_PROVINCES))).toEqual([])
    })

    it("gives each glossary term a word in every language", () => {
        const terms = GLOSSARY.terms as Record<string, string>[]
        expect(terms.flatMap((term) => LANGUAGES.filter((language) => language !== SOURCE_LANGUAGE && !term[language])
            .map((language) => `${term.en}: ${language}`))).toEqual([])
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

            if (language !== SOURCE_LANGUAGE) it("names every landmass of every board English names, and no other", () => {
                for (const [board, english] of Object.entries(ENGLISH_LANDMASSES)) {
                    if (AWAITING_TRANSLATION.has(board)) {
                        expect(LANDMASS_FILES[`./landmasses/${board}/${language}.json`], `${board} is translated: take it off AWAITING_TRANSLATION`)
                            .toBeUndefined()
                        continue
                    }
                    const names = LANDMASS_FILES[`./landmasses/${board}/${language}.json`] ?? {}
                    expect(Object.keys(names).filter((id) => !(id in english)), `${board}: ids English no longer has`).toEqual([])
                    expect(Object.keys(english).filter((id) => typeof names[id] !== "string" || !names[id]), `${board}: names missing`).toEqual([])
                }
            })

            if (language !== SOURCE_LANGUAGE) it("names every province of every board English names, and no other", () => {
                for (const [board, english] of Object.entries(ENGLISH_PROVINCES)) {
                    const names = PROVINCE_FILES[`./provinces/${board}/${language}.json`] ?? {}
                    expect(Object.keys(names).filter((id) => !(id in english)), `${board}: ids English no longer has`).toEqual([])
                    expect(Object.keys(english).filter((id) => typeof names[id] !== "string" || !names[id]), `${board}: names missing`).toEqual([])
                }
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
