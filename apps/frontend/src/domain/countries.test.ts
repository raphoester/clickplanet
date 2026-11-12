import {describe, expect, it} from "vitest"
import {Countries} from "./countries.ts"
import {ENGLISH_TRANSLATOR} from "../i18n/translator.ts"

const EMOJI = /\p{RI}|\p{Extended_Pictographic}/u

const named = () => Array.from(Countries.keys()).map((code) => ({code, name: ENGLISH_TRANSLATOR.country(code)}))

describe("Countries", () => {
    it("names a country in plain words", () => {
        expect(ENGLISH_TRANSLATOR.country("fr")).toBe("France")
        expect(ENGLISH_TRANSLATOR.country("gb-eng")).toBe("England")
        expect(ENGLISH_TRANSLATOR.country("xb")).toBe("Brittany")
    })

    it("carries no emoji in any name", () => {
        const decorated = named()
            .filter(({name}) => EMOJI.test(name))
            .map(({code, name}) => `${code}: ${name}`)
        expect(decorated).toEqual([])
    })

    it("gives every country a name of its own, with no space around it", () => {
        const wrong = named()
            .filter(({code, name}) => name === "" || name === code.toUpperCase() || name !== name.trim())
            .map(({code}) => code)
        expect(wrong).toEqual([])
    })

    it("keys every country by its own code", () => {
        const mismatched = Array.from(Countries.entries())
            .filter(([code, country]) => code !== country.code)
            .map(([code]) => code)
        expect(mismatched).toEqual([])
    })
})
