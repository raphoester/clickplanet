import {describe, expect, it} from "vitest"
import {EXPERIMENTS, variantsOf} from "./experiments.ts"

describe("the arms a clip is made in", () => {
    it("are none when the job gave none", () => {
        expect(variantsOf(null)).toEqual({})
        expect(variantsOf("")).toEqual({})
        expect(variantsOf("{}")).toEqual({})
    })

    it("are the arm of each experiment, as the job wrote them", () => {
        expect(variantsOf(`{"length": "short"}`)).toEqual({length: "short"})
        expect(variantsOf(`{"length": "as-chosen"}`)).toEqual({length: "as-chosen"})
    })

    it("refuse an experiment the generator does not know, so no clip is counted in an arm it was not made in", () => {
        expect(() => variantsOf(`{"end-card": "podium"}`)).toThrow(`no experiment "end-card"`)
        expect(() => variantsOf(`{"constructor": "short"}`)).toThrow(`no experiment "constructor"`)
    })

    it("refuse an arm the experiment does not have", () => {
        expect(() => variantsOf(`{"length": "long"}`)).toThrow(`no arm "long" in the experiment "length": as-chosen, short`)
        expect(() => variantsOf(`{"length": 8.5}`)).toThrow(`no arm 8.5`)
    })

    it("refuse what is not an object of arms", () => {
        expect(() => variantsOf(`["short"]`)).toThrow("not an object")
        expect(() => variantsOf(`null`)).toThrow("not an object")
        expect(() => variantsOf(`{length`)).toThrow()
    })

    it("name every arm in words", () => {
        for (const arms of Object.values(EXPERIMENTS)) {
            for (const arm of arms) expect(arm).toMatch(/^[a-z]+(-[a-z]+)*$/)
        }
    })
})
