import {describe, expect, it} from "vitest"
import {names, repeats} from "./words.mjs"

describe("repeats", () => {
    it("finds an answer the question already says", () => {
        expect(repeats("Mexico City is the capital of which country?", "Mexico")).toBe(true)
        expect(repeats("What is the capital of Guinea-Bissau?", "Bissau")).toBe(true)
        expect(repeats("Which continent is S.Africa in?", "Africa")).toBe(true)
    })

    it("ignores accents and case", () => {
        expect(repeats("What is the capital of Panama?", "Panamá")).toBe(true)
    })

    it("does not count a word inside another one", () => {
        expect(repeats("Which of these shares a border with Niger?", "Nigeria")).toBe(false)
    })

    it("does not count the little words", () => {
        expect(repeats("Nassau is the capital of which country?", "The Bahamas")).toBe(false)
    })
})

describe("names", () => {
    it("finds the country in the question", () => {
        expect(names("What is the capital of Bulgaria?", "Bulgaria")).toBe(true)
        expect(names("In 2025, what was Bulgaria's currency?", "Bulgaria")).toBe(true)
        expect(names("Which of these shares a border with UK?", "UK")).toBe(true)
    })

    it("does not find it inside another name", () => {
        expect(names("Which of these shares a border with Nigeria?", "Niger")).toBe(false)
    })

    it("does not find it where the question does not say it", () => {
        expect(names("Sofia is the capital of which country?", "Bulgaria")).toBe(false)
    })
})
