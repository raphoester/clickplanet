import {describe, expect, it} from "vitest"
import {EVERY_WORD_MASKED, NO_FILTER, wordFilterOf} from "./wordFilter.ts"

describe("wordFilterOf", () => {
    const filter = wordFilterOf(["ass", "FUCK", "a$$", "!njun", "pain in the ass", "fk", "", "  "])

    it("masks the letters between the first and the last", () => {
        expect(filter.masked("you ass")).toBe("you a*s")
        expect(filter.masked("fuck off")).toBe("f**k off")
    })

    it("matches whole words only", () => {
        expect(filter.masked("there is an assassin")).toBe("there is an assassin")
        expect(filter.masked("classy")).toBe("classy")
    })

    it("ignores the case and keeps it", () => {
        expect(filter.masked("FuCk")).toBe("F**k")
    })

    it("leaves the punctuation around a word", () => {
        expect(filter.masked("(ass)!")).toBe("(a*s)!")
        expect(filter.masked("fuck, fuck.")).toBe("f**k, f**k.")
    })

    it("matches an entry that has symbols", () => {
        expect(filter.masked("a$$ hole")).toBe("a*$ hole")
        expect(filter.masked("!njun?")).toBe("!***n?")
    })

    it("masks every word of a phrase", () => {
        expect(filter.masked("a pain in the ass!")).toBe("a p**n i* t*e a*s!")
        expect(filter.masked("pain  in\nthe ass")).toBe("p**n  i*\nt*e a*s")
        expect(filter.masked("a pain in the neck")).toBe("a pain in the neck")
    })

    it("masks a short word and keeps a letter it can", () => {
        expect(filter.masked("fk")).toBe("f*")
        expect(wordFilterOf(["x"]).masked("x")).toBe("*")
    })

    it("counts a letter outside the basic plane as one", () => {
        expect(wordFilterOf(["𝒶𝓈𝓈"]).masked("𝒶𝓈𝓈")).toBe("𝒶*𝓈")
    })

    it("leaves a text with nothing to mask as it is", () => {
        expect(filter.masked("")).toBe("")
        expect(filter.masked("  hello   world ")).toBe("  hello   world ")
    })
})

describe("NO_FILTER", () => {
    it("masks nothing", () => {
        expect(NO_FILTER.masked("fuck")).toBe("fuck")
    })
})

describe("EVERY_WORD_MASKED", () => {
    it("masks every word", () => {
        expect(EVERY_WORD_MASKED.masked("hello my friend")).toBe("h***o m* f****d")
    })
})
