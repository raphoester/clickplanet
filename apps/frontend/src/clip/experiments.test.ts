import {describe, expect, it} from "vitest"
import {EXPERIMENTS, variantsOf} from "./experiments.ts"

describe("the arms a clip is made in", () => {
    it("are none when the job gave none", () => {
        expect(variantsOf(null)).toEqual({})
        expect(variantsOf("")).toEqual({})
        expect(variantsOf("{}")).toEqual({})
    })

    it("refuse an experiment the generator does not know, so no clip is counted in an arm it was not made in", () => {
        expect(() => variantsOf(`{"end-card": "podium"}`)).toThrow(`no experiment "end-card"`)
        expect(() => variantsOf(`{"length": "short"}`)).toThrow(`no experiment "length"`)
        expect(() => variantsOf(`{"sound": "trending"}`)).toThrow(`no experiment "sound"`)
        expect(() => variantsOf(`{"constructor": "short"}`)).toThrow(`no experiment "constructor"`)
    })

    it("refuse what is not an object of arms", () => {
        expect(() => variantsOf(`["short"]`)).toThrow("not an object")
        expect(() => variantsOf(`null`)).toThrow("not an object")
        expect(() => variantsOf(`{length`)).toThrow()
    })

    it("name every arm in words", () => {
        for (const arms of Object.values<readonly string[]>(EXPERIMENTS)) {
            for (const arm of arms) expect(arm).toMatch(/^[a-z]+(-[a-z]+)*$/)
        }
    })
})
